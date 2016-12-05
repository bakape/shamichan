// Renders various HTML forms

package templates

import (
	"bytes"
	"reflect"
	"strings"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

type boardList struct {
	Boards config.BoardTitles
	Lang   lang.Pack
}

type formSpecs struct {
	Specs []inputSpec
	Lang  lang.Pack
}

// BoardNavigation renders a board selection and search form
func BoardNavigation(ln lang.Pack) ([]byte, error) {
	return exec("boardNavigation", boardList{
		Boards: config.GetBoardTitles(),
		Lang:   ln,
	})
}

// Execute a template by id with the provided variables
func exec(id string, vars interface{}) ([]byte, error) {
	var w bytes.Buffer
	err := tmpl[id].Execute(&w, vars)
	return w.Bytes(), err
}

// CreateBoard renders a the form for creating new boards
func CreateBoard(ln lang.Pack) ([]byte, error) {
	return exec("createBoard", struct {
		Captcha bool
		formSpecs
	}{
		Captcha: config.Get().Captcha,
		formSpecs: formSpecs{
			Specs: specs["createBoard"],
			Lang:  ln,
		},
	})
}

// ConfigureBoard renders a form for setting board configurations
func ConfigureBoard(conf config.BoardConfigs, ln lang.Pack) ([]byte, error) {
	return configurationTable(reflect.ValueOf(conf), "configureBoard", ln)
}

func configurationTable(v reflect.Value, key string, ln lang.Pack) (
	[]byte, error,
) {
	// Copy over all spec structs, so the mutations don't affect them
	noValues := specs[key]
	withValues := make([]inputSpec, len(noValues))
	copy(withValues, noValues)

	// Assign values to all specs
	for i, s := range withValues {
		withValues[i].Val = v.FieldByName(strings.Title(s.ID)).Interface()
	}

	return exec("tableForm", formSpecs{
		Specs: withValues,
		Lang:  ln,
	})
}

// ConfigureServer renders the form for changing server configurations
func ConfigureServer(conf config.Configs, ln lang.Pack) ([]byte, error) {
	return configurationTable(reflect.ValueOf(conf), "configureServer", ln)
}

// ChangePassword renders a form for changing an account's password
func ChangePassword(ln lang.Pack) ([]byte, error) {
	return exec("tableForm", formSpecs{
		Specs: specs["changePassword"],
		Lang:  ln,
	})
}
