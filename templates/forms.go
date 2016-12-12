// Renders various HTML forms

package templates

import (
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

// ConfigureBoard renders a form for setting board configurations
func ConfigureBoard(conf config.BoardConfigs, ln lang.Pack) string {
	v := reflect.ValueOf(conf)
	return configurationTable(v, "configureBoard", false, ln)
}

func configurationTable(
	v reflect.Value,
	key string,
	needCaptcha bool,
	ln lang.Pack,
) string {
	// Copy over all spec structs, so the mutations don't affect them
	noValues := specs[key]
	withValues := make([]inputSpec, len(noValues))
	copy(withValues, noValues)

	// Assign values to all specs
	for i, s := range withValues {
		withValues[i].Val = v.FieldByName(strings.Title(s.ID)).Interface()
	}

	return tableForm(withValues, needCaptcha, ln)
}

// ConfigureServer renders the form for changing server configurations
func ConfigureServer(conf config.Configs, ln lang.Pack) string {
	v := reflect.ValueOf(conf)
	return configurationTable(v, "configureServer", false, ln)
}

// ChangePassword renders a form for changing an account's password
func ChangePassword(ln lang.Pack) string {
	return tableForm(specs["changePassword"], true, ln)
}
