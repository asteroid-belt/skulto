package views

import (
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestResetViewStartsAsync(t *testing.T) {
	// Create reset view with mock config
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.Init()

	// Select reset button
	rv.selected = true

	// Press enter - should start async, not block
	start := time.Now()
	back, _, cmd := rv.Update("enter")
	elapsed := time.Since(start)

	assert.False(t, back, "should not go back immediately")
	assert.NotNil(t, cmd, "should return a command")
	assert.True(t, rv.resetting, "should be in resetting state")
	assert.Less(t, elapsed, 100*time.Millisecond, "should return immediately")
}

func TestResetViewHandlesCompletion(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.resetting = true

	// Simulate completion message (success case is handled by app.go now)
	rv.HandleResetComplete(ResetCompleteMsg{Success: true, Err: nil, NewDB: nil})

	assert.False(t, rv.resetting, "should no longer be resetting")
	assert.True(t, rv.confirmed, "should be confirmed")
	assert.Empty(t, rv.error, "should have no error")
}

func TestResetViewHandlesCompletionWithError(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.resetting = true

	// Simulate completion with error
	rv.HandleResetComplete(ResetCompleteMsg{Success: false, Err: assert.AnError, NewDB: nil})

	assert.False(t, rv.resetting, "should no longer be resetting")
	assert.True(t, rv.confirmed, "should be confirmed")
	assert.NotEmpty(t, rv.error, "should have error message")
}

func TestResetViewIgnoresKeysWhileResetting(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.resetting = true

	// Try pressing various keys while resetting
	back, success, cmd := rv.Update("enter")
	assert.False(t, back)
	assert.False(t, success)
	assert.Nil(t, cmd)

	back, success, cmd = rv.Update("esc")
	assert.False(t, back)
	assert.False(t, success)
	assert.Nil(t, cmd)

	back, success, cmd = rv.Update("left")
	assert.False(t, back)
	assert.False(t, success)
	assert.Nil(t, cmd)
}

func TestResetViewCancelDoesNotTriggerAsync(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.Init()

	// Cancel is selected by default
	assert.False(t, rv.selected)

	// Press enter on cancel - should go back immediately without async
	back, success, cmd := rv.Update("enter")

	assert.True(t, back, "should go back")
	assert.False(t, success, "reset was not successful (cancelled)")
	assert.Nil(t, cmd, "should not return a command")
	assert.False(t, rv.resetting, "should not be resetting")
}

func TestResetViewEscapeGoesBack(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.Init()

	back, success, cmd := rv.Update("esc")

	assert.True(t, back, "should go back")
	assert.False(t, success, "reset was not successful")
	assert.Nil(t, cmd, "should not return a command")
}

func TestResetViewResettingViewRenders(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.SetSize(80, 24)
	rv.resetting = true

	view := rv.View()

	assert.Contains(t, view, "Resetting")
	assert.Contains(t, view, "Please wait")
}

func TestResetViewButtonNavigation(t *testing.T) {
	cfg := &config.Config{BaseDir: t.TempDir()}
	rv := NewResetView(nil, cfg)
	rv.Init()

	// Initially cancel is selected
	assert.False(t, rv.selected)

	// Navigate right to reset
	rv.Update("right")
	assert.True(t, rv.selected)

	// Navigate left to cancel
	rv.Update("left")
	assert.False(t, rv.selected)

	// Vim bindings
	rv.Update("l")
	assert.True(t, rv.selected)

	rv.Update("h")
	assert.False(t, rv.selected)
}
