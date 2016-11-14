// Package config stores and exports the configuration for server-side use and
// the public availability JSON struct, which includes a small subset of the
// server configuration.
package config

import (
	"encoding/json"
	"sync"
	"time"

	"sort"

	"github.com/bakape/meguca/util"
)

var (
	// Ensures no reads happen, while the configuration is reloading
	globalMu, boardMu sync.RWMutex

	// Contains currently loaded global server configuration
	global *Configs

	// Map of board IDs to their configuration structs
	boardConfigs = map[string]BoardConfContainer{}

	// AllBoardConfigs stores board-specific configurations for the /all/
	// metaboard. Constant.
	AllBoardConfigs = BoardConfContainer{
		BoardConfigs: BoardConfigs{
			ID: "all",
			BoardPublic: BoardPublic{
				PostParseConfigs: PostParseConfigs{
					HashCommands: true,
				},
				Spoilers: true,
				CodeTags: true,
				Spoiler:  "default.jpg",
				Title:    "Aggregator metaboard",
				Banners:  []string{},
			},
		},
		Hash: "0",
	}

	// JSON of client-accessible configuration
	clientJSON []byte

	// Hash of the global configs. Used for live reloading configuration on the
	// client.
	hash string

	// AllowedOrigin stores the accepted client origin for websocket and file
	// upload requests. Set only on server start.
	AllowedOrigin string

	// Defaults contains the default server configuration values
	Defaults = Configs{
		ThreadExpiry:  14,
		BoardExpiry:   7,
		JPEGQuality:   80,
		PNGQuality:    20,
		MaxSize:       5,
		MaxHeight:     6000,
		MaxWidth:      6000,
		SessionExpiry: 30,
		Salt:          "LALALALALALALALALALALALALALALALALALALALA",
		FeedbackEmail: "admin@email.com",
		Public: Public{
			DefaultCSS:  "moe",
			FAQ:         defaultFAQ,
			DefaultLang: "en_GB",
			Links:       map[string]string{"4chan": "http://www.4chan.org/"},
		},
	}

	// EightballDefaults contains the default eightball answer set
	EightballDefaults = []string{
		"Yes",
		"No",
		"Maybe",
		"It can't be helped",
		"Hell yeah, motherfucker!",
		"Anta baka?",
	}
)

// Default string for the FAQ panel
const defaultFAQ = `Supported upload file types are JPEG, PNG, APNG, WEBM, MP3, MP4 and OGG.
Encase words in ** to spoiler them. Spoilers reset on newline.
<hr>Hash commands:
#d100 #2d100 - Roll dice
#flip - Coin flip
#8ball - An 8ball

All hash commands need to be input on their own line`

// Configs stores the global server configuration
type Configs struct {
	Public
	PruneThreads      bool   `json:"pruneThreads" gorethink:"pruneThreads"`
	PruneBoards       bool   `json:"pruneBoards" gorethink:"pruneBoards"`
	Pyu               bool   `json:"pyu" gorethink:"pyu"`
	MaxWidth          uint16 `json:"maxWidth" gorethink:"maxWidth"`
	MaxHeight         uint16 `json:"maxHeight" gorethink:"maxHeight"`
	JPEGQuality       int
	PNGQuality        int
	ThreadExpiry      uint          `json:"threadExpiry" gorethink:"threadExpiry"`
	BoardExpiry       uint          `json:"boardExpiry" gorethink:"boardExpiry"`
	MaxSize           int64         `json:"maxSize" gorethink:"maxSize"`
	Salt              string        `json:"salt" gorethink:"salt"`
	FeedbackEmail     string        `json:"feedbackEmail" gorethink:"feedbackEmail"`
	CaptchaPrivateKey string        `json:"captchaPrivateKey" gorethink:"captchaPrivateKey"`
	SessionExpiry     time.Duration `json:"sessionExpiry" gorethink:"sessionExpiry"`
}

// Public contains configurations exposeable through public availability APIs
type Public struct {
	Hats             bool   `json:"hats" gorethink:"hats"`
	Captcha          bool   `json:"captcha" gorethink:"captcha"`
	Mature           bool   `json:"mature" gorethink:"mature"`
	DefaultLang      string `json:"defaultLang" gorethink:"defaultLang"`
	DefaultCSS       string `json:"defaultCSS" gorethink:"defaultCSS"`
	CaptchaPublicKey string `json:"captchaPublicKey" gorethink:"captchaPublicKey"`
	FAQ              string
	Links            map[string]string `json:"links" gorethink:"links"`
}

// BoardConfigs stores board-specific configuration
type BoardConfigs struct {
	BoardPublic
	ID        string              `json:"id" gorethink:"id"`
	Eightball []string            `json:"eightball" gorethink:"eightball"`
	Staff     map[string][]string `json:"staff" gorethink:"staff"`
}

// BoardPublic contains publically accessible board-specific configurations
type BoardPublic struct {
	PostParseConfigs
	Spoilers bool     `json:"spoilers" gorethink:"spoilers"`
	CodeTags bool     `json:"codeTags" gorethink:"codeTags"`
	Spoiler  string   `json:"spoiler" gorethink:"spoiler"`
	Title    string   `json:"title" gorethink:"title"`
	Notice   string   `json:"notice" gorethink:"notice"`
	Rules    string   `json:"rules" gorethink:"rules"`
	Banners  []string `json:"banners" gorethink:"banners"`
}

// BoardConfContainer contains configurations for an individual board as well
// as pregenerated public JSON and it's hash
type BoardConfContainer struct {
	BoardConfigs
	JSON []byte
	Hash string
}

// DatabaseBoardConfigs contains extra fields not exposed on database reads
type DatabaseBoardConfigs struct {
	BoardConfigs
	Created time.Time `gorethink:"created"`
}

// PostParseConfigs contains board-specific flags for post text parsing
type PostParseConfigs struct {
	ReadOnly     bool `json:"readOnly" gorethink:"readOnly"`
	TextOnly     bool `json:"textOnly" gorethink:"textOnly"`
	ForcedAnon   bool `json:"forcedAnon" gorethink:"forcedAnon"`
	HashCommands bool `json:"hashCommands" gorethink:"hashCommands"`
}

// BoardTitle contains a board's ID and title
type BoardTitle struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// BoardTitles implements sort.Interface
type BoardTitles []BoardTitle

// Generate /all/ board configs
func init() {
	var err error
	AllBoardConfigs.JSON, err = json.Marshal(AllBoardConfigs.BoardPublic)
	if err != nil {
		panic(err)
	}
}

func (b BoardTitles) Len() int {
	return len(b)
}

func (b BoardTitles) Less(i, j int) bool {
	return b[i].ID < b[j].ID
}

func (b BoardTitles) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// Get returns a pointer to the current server configuration struct. Callers
// should not modify this struct.
func Get() *Configs {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// Set sets the internal configuration struct
func Set(c Configs) error {
	client, err := json.Marshal(c.Public)
	if err != nil {
		return err
	}
	h := util.HashBuffer(client)

	globalMu.Lock()
	clientJSON = client
	global = &c
	hash = h
	globalMu.Unlock()

	return nil
}

// GetClient returns public availability configuration JSON and a truncated
// configuration MD5 hash
func GetClient() ([]byte, string) {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return clientJSON, hash
}

// SetClient sets the client configuration JSON and hash. To be used only in
// tests.
func SetClient(json []byte, cHash string) {
	globalMu.Lock()
	clientJSON = json
	hash = cHash
	globalMu.Unlock()
}

// GetBoardConfigs returns board-specific configurations for a board combined
// with pregenerated public JSON of these configurations and their hash
func GetBoardConfigs(b string) BoardConfContainer {
	if b == "all" {
		return AllBoardConfigs
	}
	boardMu.RLock()
	defer boardMu.RUnlock()
	return boardConfigs[b]
}

// GetBoardTitles returns a slice of all existing boards and their titles
func GetBoardTitles() BoardTitles {
	boardMu.RLock()
	defer boardMu.RUnlock()

	bt := make(BoardTitles, 0, len(boardConfigs))
	for id, conf := range boardConfigs {
		bt = append(bt, BoardTitle{
			ID:    id,
			Title: conf.Title,
		})
	}

	sort.Sort(bt)
	return bt
}

// GetBoards returns an array of currently existing boards
func GetBoards() []string {
	boardMu.RLock()
	defer boardMu.RUnlock()
	boards := make([]string, 0, len(boardConfigs))
	for b := range boardConfigs {
		boards = append(boards, b)
	}
	return boards
}

// IsBoard returns whether the passed string is a currently existing board
func IsBoard(b string) bool {
	boardMu.RLock()
	defer boardMu.RUnlock()
	_, ok := boardConfigs[b]
	return ok
}

// SetBoardConfigs sets configurations for a specific board as well as
// pregenerates their public JSON and hash. Returns if any changes were made to
// the configs in result.
func SetBoardConfigs(conf BoardConfigs) (bool, error) {
	cont := BoardConfContainer{
		BoardConfigs: conf,
	}
	var err error
	cont.JSON, err = json.Marshal(conf.BoardPublic)
	if err != nil {
		return false, err
	}
	cont.Hash = util.HashBuffer(cont.JSON)

	boardMu.Lock()
	defer boardMu.Unlock()

	// Nothing changed
	if boardConfigs[conf.ID].Hash == cont.Hash {
		return false, nil
	}

	// Swap config
	boardConfigs[conf.ID] = cont
	return true, nil
}

// RemoveBoard removes a board from the exiting board list and deletes its
// configurations. To be called, when a board is deleted.
func RemoveBoard(b string) {
	boardMu.Lock()
	defer boardMu.Unlock()

	delete(boardConfigs, b)
}

// Clear resets package state. Only use in tests.
func Clear() {
	boardMu.Lock()
	defer boardMu.Unlock()
	globalMu.RLock()
	defer globalMu.RUnlock()

	global = &Configs{}
	boardConfigs = map[string]BoardConfContainer{}
	clientJSON = nil
	hash = ""
}

// ClearBoards clears any existing board configuration entries. Only use in
// tests.
func ClearBoards() {
	boardMu.Lock()
	defer boardMu.Unlock()

	boardConfigs = map[string]BoardConfContainer{}
}
