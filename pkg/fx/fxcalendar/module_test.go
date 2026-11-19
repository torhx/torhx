package fxcalendar_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/torhx/torhx/pkg/component/calendar"
	"github.com/torhx/torhx/pkg/fx/fxcalendar"
)

func TestFxCalendarModuleProvidesTheBuiltInCalendars(t *testing.T) {
	t.Parallel()

	var registry calendar.Registry

	app := fxtest.New(t, fx.NopLogger, fxcalendar.FxCalendarModule, fx.Populate(&registry))
	app.RequireStart()
	t.Cleanup(app.RequireStop)

	id, err := calendar.NewID("XNYS")
	require.NoError(t, err)

	xnys, err := registry.Get(id)
	require.NoError(t, err)
	assert.Equal(t, "XNYS", xnys.Exchange().Code().String())
}

func TestFxCalendarModuleNamesCalendarsByExchangeCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		wantErr string
	}{
		{name: "exchange with calendar data", code: "XNYS"},
		{name: "exchange without calendar data", code: "XTKS", wantErr: "no trading calendar XTKS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got calendar.TradingCalendar

			app := fx.New(
				fx.NopLogger,
				fxcalendar.FxCalendarModule,
				fx.Invoke(fx.Annotate(func(c calendar.TradingCalendar) { got = c }, fx.ParamTags(`name:"`+tt.code+`"`))),
			)
			if tt.wantErr != "" {
				require.ErrorContains(t, app.Err(), tt.wantErr)

				return
			}

			require.NoError(t, app.Err())
			assert.Equal(t, tt.code, got.ID().String())
		})
	}
}
