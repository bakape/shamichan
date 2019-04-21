// Renders various HTML forms

package templates

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/bakape/meguca/config"
)

// ConfigureBoard renders a form for setting board configurations
func ConfigureBoard(w io.Writer, conf config.BoardConfigs) {
	configurationTable(w, reflect.ValueOf(conf), "configureBoard", true)
}

func configurationTable(w io.Writer, v reflect.Value, key string,
	needCaptcha bool,
) {
	// Copy over all spec structs, so the mutations don't affect them
	noValues := specs[key]
	withValues := make([]inputSpec, len(noValues))
	copy(withValues, noValues)

	// Assign values to all specs
	for i, s := range withValues {
		key := strings.Title(s.ID)
		if key == "" { // <hr>
			continue
		}
		v := v.FieldByName(key)
		if !v.IsValid() {
			// Programmer error. Should not happen in production.
			panic(fmt.Errorf("struct key not found: %s", key))
		}
		switch k := v.Kind(); k {
		case reflect.Uint8, reflect.Uint16:
			v = v.Convert(reflect.TypeOf(uint(0)))
		}
		withValues[i].Val = v.Interface()
	}

	writetableForm(w, withValues, needCaptcha)
}

// ConfigureServer renders the form for changing server configurations
func ConfigureServer(w io.Writer, conf config.Configs) {
	configurationTable(w, reflect.ValueOf(conf), "configureServer", false)
}

// ChangePassword renders a form for changing an account's password
func ChangePassword(w io.Writer) {
	writetableForm(w, specs["changePassword"], true)
}

// StaffAssignment renders a staff assignment form with the current staff
// already filled in
func StaffAssignment(w io.Writer, staff [3][]string) {
	var specs [3]inputSpec
	for i, id := range [3]string{"owners", "moderators", "janitors"} {
		sort.Strings(staff[i])
		specs[i] = inputSpec{
			ID:   id,
			Type: _array,
			Val:  staff[i],
		}
	}

	writetableForm(w, specs[:], true)
}
