package unit

import (
	"strings"
	"testing"

	"github.com/inoki/sgreen/internal/session"
)

func TestCurrentUser(t *testing.T) {
	user := session.CurrentUser()

	if user == "" {
		t.Logf("CurrentUser returned empty - may be expected in some environments")
	}

	if strings.Contains(user, " ") {
		t.Errorf("Username should not contain spaces: %q", user)
	}
}

func TestSessionConfigDefaults(t *testing.T) {
	config := &session.Config{}

	if config.Term != "" {
		t.Errorf("Default Term should be empty")
	}
	if config.Scrollback != 0 {
		t.Errorf("Default Scrollback should be 0")
	}
	if config.UTF8 {
		t.Errorf("Default UTF8 should be false")
	}
}

func TestSessionConfigFields(t *testing.T) {
	config := &session.Config{
		Term:       "xterm-256color",
		UTF8:       true,
		Scrollback: 1000,
		Encoding:   "UTF-8",
	}

	if config.Term != "xterm-256color" {
		t.Errorf("Term field not set correctly")
	}
	if !config.UTF8 {
		t.Errorf("UTF8 field not set correctly")
	}
	if config.Scrollback != 1000 {
		t.Errorf("Scrollback field not set correctly")
	}
	if config.Encoding != "UTF-8" {
		t.Errorf("Encoding field not set correctly")
	}
}

func TestSessionConfigWithEmptyEncoding(t *testing.T) {
	config := &session.Config{
		Term:       "xterm",
		UTF8:       false,
		Scrollback: 500,
		Encoding:   "",
	}

	if config.Encoding != "" {
		t.Errorf("Empty encoding should remain empty")
	}
	if config.Scrollback != 500 {
		t.Errorf("Scrollback should be 500")
	}
}

func TestSessionConfigWithNegativeScrollback(t *testing.T) {
	config := &session.Config{
		Scrollback: -100,
	}

	if config.Scrollback != -100 {
		t.Errorf("Negative scrollback should be preserved")
	}
}

func TestSessionConfigWithLargeValues(t *testing.T) {
	config := &session.Config{
		Scrollback: 100000,
	}

	if config.Scrollback != 100000 {
		t.Errorf("Large scrollback value should be preserved")
	}
}