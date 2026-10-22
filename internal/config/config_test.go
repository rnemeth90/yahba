package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetupProxy(t *testing.T) {

}

func TestSkipNameResolution(t *testing.T) {

}

func TestSetupCustomResolver(t *testing.T) {

}

func TestValidateModes(t *testing.T) {
	base := func() Config {
		return Config{URL: "http://localhost", Method: "GET", Timeout: 5, RPS: 1, Requests: 1}
	}

	c := base()
	c.RPS = MaxRPS + 1
	assert.ErrorIs(t, c.Validate(), ErrRPSTooHigh)

	c = base()
	c.Duration = 5 * time.Second
	c.Requests, c.RPS = 0, 0
	assert.Equal(t, ModeMax, c.EffectiveMode())
	assert.NoError(t, c.Validate())

	c = base()
	c.Mode = ModeMax
	assert.ErrorIs(t, c.Validate(), ErrInvalidDuration)

	c = base()
	c.Mode = "bogus"
	assert.ErrorIs(t, c.Validate(), ErrInvalidMode)

	ramp := func() Config {
		c := base()
		c.Mode = ModeRamp
		c.RampStart, c.RampStep = 10, 10
		c.StepDuration = time.Second
		c.MaxErrorRate = 5
		return c
	}
	c = ramp()
	assert.NoError(t, c.Validate())

	c = ramp()
	c.RampStep = 0
	assert.ErrorIs(t, c.Validate(), ErrInvalidRamp)

	c = ramp()
	c.RampMax = 5
	assert.ErrorIs(t, c.Validate(), ErrInvalidRamp)

	c = ramp()
	c.RampMax = MaxRPS + 1
	assert.ErrorIs(t, c.Validate(), ErrRPSTooHigh)

	c = ramp()
	c.MaxErrorRate = 0
	assert.ErrorIs(t, c.Validate(), ErrInvalidRampThreshold)
}
