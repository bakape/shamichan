// Package config stores and exports the configuration for server-side use and
// the public availability JSON struct, which includes a small subset of the
// server configuration.
package config

import (
	"encoding/json"
	"reflect"
	"sort"
	"sync"

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
				CodeTags: true,
				Title:    "Aggregator metaboard",
			},
		},
		Hash: "0",
	}

	// JSON of client-accessible configuration
	clientJSON []byte

	// Hash of the global configs. Used for live reloading configuration on the
	// client.
	hash string

	// Defaults contains the default server configuration values
	Defaults = Configs{
		ThreadExpiry:  14,
		BoardExpiry:   7,
		JPEGQuality:   80,
		MaxSize:       5,
		MaxHeight:     6000,
		MaxWidth:      6000,
		SessionExpiry: 30,
		Salt:          "LALALALALALALALALALALALALALALALALALALALA",
		FeedbackEmail: "admin@email.com",
		RootURL:       "http://localhost",
		FAQ:           defaultFAQ,
		Public: Public{
			DefaultCSS:  "moe",
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
const defaultFAQ = `Supported upload file types are JPEG, PNG, APNG, WEBM, MP3, MP4, OGG, PDF, ZIP, 7Z, TAR.GZ and TAR.XZ.
Encase text in ** to spoiler and in ` + "``" + ` to highlight programing code syntax.
<hr>Hash commands:
#d100 #2d100 - Roll dice
#flip - Coin flip
#8ball - An 8ball`

// Configs stores the global server configuration
type Configs struct {
	Public
	PruneThreads      bool `json:"pruneThreads"`
	PruneBoards       bool `json:"pruneBoards"`
	Pyu               bool `json:"pyu"`
	JPEGQuality       uint8
	MaxWidth          uint16 `json:"maxWidth"`
	MaxHeight         uint16 `json:"maxHeight"`
	ThreadExpiry      uint   `json:"threadExpiry"`
	BoardExpiry       uint   `json:"boardExpiry"`
	MaxSize           uint   `json:"maxSize"`
	SessionExpiry     uint   `json:"sessionExpiry"`
	RootURL           string `json:"rootURL"`
	Salt              string `json:"salt"`
	FeedbackEmail     string `json:"feedbackEmail"`
	CaptchaPrivateKey string `json:"captchaPrivateKey"`
	FAQ               string
}

// Public contains configurations exposeable through public availability APIs
type Public struct {
	Captcha           bool              `json:"captcha"`
	Mature            bool              `json:"mature"`
	DefaultLang       string            `json:"defaultLang"`
	DefaultCSS        string            `json:"defaultCSS"`
	CaptchaPublicKey  string            `json:"captchaPublicKey"`
	ImageRootOverride string            `json:"imageRootOverride"`
	Links             map[string]string `json:"links"`
}

// BoardConfigs stores board-specific configuration
type BoardConfigs struct {
	BoardPublic
	ID        string   `json:"id"`
	Eightball []string `json:"eightball"`
}

// BoardPublic contains publically accessible board-specific configurations
type BoardPublic struct {
	PostParseConfigs
	CodeTags bool   `json:"codeTags"`
	Title    string `json:"title"`
	Notice   string `json:"notice"`
	Rules    string `json:"rules"`
}

// BoardConfContainer contains configurations for an individual board as well
// as pregenerated public JSON and it's hash
type BoardConfContainer struct {
	BoardConfigs
	JSON []byte
	Hash string
}

// PostParseConfigs contains board-specific flags for post text parsing
type PostParseConfigs struct {
	ReadOnly     bool `json:"readOnly"`
	TextOnly     bool `json:"textOnly"`
	ForcedAnon   bool `json:"forcedAnon"`
	HashCommands bool `json:"hashCommands"`
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
// with pregenerated public JSON of these configurations and their hash. Do
// not modify the retrieved struct.
func GetBoardConfigs(b string) BoardConfContainer {
	if b == "all" {
		return AllBoardConfigs
	}
	boardMu.RLock()
	defer boardMu.RUnlock()
	return boardConfigs[b]
}

// GetAllBoardConfigs returns board-specific configurations for all boards. Do
// not modify the retrieved structs.
func GetAllBoardConfigs() []BoardConfContainer {
	boardMu.RLock()
	defer boardMu.RUnlock()

	conf := make([]BoardConfContainer, 0, len(boardConfigs))
	for _, c := range boardConfigs {
		conf = append(conf, c)
	}
	return conf
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
	sort.Strings(boards)
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
	noChange := reflect.DeepEqual(
		boardConfigs[conf.ID].BoardConfigs,
		cont.BoardConfigs,
	)
	if noChange {
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
	globalMu.RLock()
	defer globalMu.RUnlock()

	global = &Configs{}
	clientJSON = nil
	hash = ""

	ClearBoards()
}

// ClearBoards clears any existing board configuration entries. Only use in
// tests.
func ClearBoards() {
	boardMu.Lock()
	defer boardMu.Unlock()

	boardConfigs = map[string]BoardConfContainer{}
}
