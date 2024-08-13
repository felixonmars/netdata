// SPDX-License-Identifier: GPL-3.0-or-later

package uwsgi

import (
	"errors"
	"os"
	"testing"

	"github.com/netdata/netdata/go/plugins/plugin/go.d/agent/module"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	dataConfigJSON, _ = os.ReadFile("testdata/config.json")
	dataConfigYAML, _ = os.ReadFile("testdata/config.yaml")

	dataStats, _          = os.ReadFile("testdata/stats.json")
	dataStatsNoWorkers, _ = os.ReadFile("testdata/stats_no_workers.json")
)

func Test_testDataIsValid(t *testing.T) {
	for name, data := range map[string][]byte{
		"dataConfigJSON":     dataConfigJSON,
		"dataConfigYAML":     dataConfigYAML,
		"dataStats":          dataStats,
		"dataStatsNoWorkers": dataStatsNoWorkers,
	} {
		require.NotNil(t, data, name)
	}
}

func TestUwsgi_ConfigurationSerialize(t *testing.T) {
	module.TestConfigurationSerialize(t, &Uwsgi{}, dataConfigJSON, dataConfigYAML)
}

func TestUwsgi_Init(t *testing.T) {
	tests := map[string]struct {
		config   Config
		wantFail bool
	}{
		"success with default config": {
			wantFail: false,
			config:   New().Config,
		},
		"fails if address not set": {
			wantFail: true,
			config: func() Config {
				conf := New().Config
				conf.Address = ""
				return conf
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			uw := New()
			uw.Config = test.config

			if test.wantFail {
				assert.Error(t, uw.Init())
			} else {
				assert.NoError(t, uw.Init())
			}
		})
	}
}

func TestUwsgi_Cleanup(t *testing.T) {
	tests := map[string]struct {
		prepare func() *Uwsgi
	}{
		"not initialized": {
			prepare: func() *Uwsgi {
				return New()
			},
		},
		"after check": {
			prepare: func() *Uwsgi {
				uw := New()
				uw.newConn = func(config Config) uwsgiConn { return prepareMockOk() }
				_ = uw.Check()
				return uw
			},
		},
		"after collect": {
			prepare: func() *Uwsgi {
				uw := New()
				uw.newConn = func(config Config) uwsgiConn { return prepareMockOk() }
				_ = uw.Collect()
				return uw
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			uw := test.prepare()

			assert.NotPanics(t, uw.Cleanup)
		})
	}
}

func TestUwsgi_Charts(t *testing.T) {
	assert.NotNil(t, New().Charts())
}

func TestUwsgi_Check(t *testing.T) {
	tests := map[string]struct {
		prepareMock func() *mockUwsgiConn
		wantFail    bool
	}{
		"success case": {
			wantFail:    false,
			prepareMock: prepareMockOk,
		},
		"success case no workers": {
			wantFail:    false,
			prepareMock: prepareMockOkNoWorkers,
		},
		"err on connect": {
			wantFail:    true,
			prepareMock: prepareMockErrOnConnect,
		},
		"unexpected response": {
			wantFail:    true,
			prepareMock: prepareMockUnexpectedResponse,
		},
		"empty response": {
			wantFail:    true,
			prepareMock: prepareMockEmptyResponse,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			uw := New()
			mock := test.prepareMock()
			uw.newConn = func(config Config) uwsgiConn { return mock }

			if test.wantFail {
				assert.Error(t, uw.Check())
			} else {
				assert.NoError(t, uw.Check())
			}
		})
	}
}

func TestUwsgi_Collect(t *testing.T) {
	tests := map[string]struct {
		prepareMock             func() *mockUwsgiConn
		wantMetrics             map[string]int64
		wantCharts              int
		disconnectBeforeCleanup bool
		disconnectAfterCleanup  bool
	}{
		"success case": {
			prepareMock:             prepareMockOk,
			wantCharts:              len(charts) + len(workerChartsTmpl)*2,
			disconnectBeforeCleanup: true,
			disconnectAfterCleanup:  true,
			wantMetrics: map[string]int64{
				"worker_1_average_request_time":                  1,
				"worker_1_delta_requests":                        1,
				"worker_1_exceptions":                            1,
				"worker_1_harakiris":                             1,
				"worker_1_memory_rss":                            1,
				"worker_1_memory_vsz":                            1,
				"worker_1_request_handling_status_accepting":     1,
				"worker_1_request_handling_status_not_accepting": 0,
				"worker_1_requests":                              1,
				"worker_1_respawns":                              1,
				"worker_1_status_busy":                           0,
				"worker_1_status_cheap":                          0,
				"worker_1_status_idle":                           1,
				"worker_1_status_pause":                          0,
				"worker_1_status_sig":                            0,
				"worker_1_tx":                                    1,
				"worker_2_average_request_time":                  1,
				"worker_2_delta_requests":                        1,
				"worker_2_exceptions":                            1,
				"worker_2_harakiris":                             1,
				"worker_2_memory_rss":                            1,
				"worker_2_memory_vsz":                            1,
				"worker_2_request_handling_status_accepting":     1,
				"worker_2_request_handling_status_not_accepting": 0,
				"worker_2_requests":                              1,
				"worker_2_respawns":                              1,
				"worker_2_status_busy":                           0,
				"worker_2_status_cheap":                          0,
				"worker_2_status_idle":                           1,
				"worker_2_status_pause":                          0,
				"worker_2_status_sig":                            0,
				"worker_2_tx":                                    1,
				"workers_exceptions":                             2,
				"workers_harakiris":                              2,
				"workers_requests":                               2,
				"workers_respawns":                               2,
				"workers_tx":                                     2,
			},
		},
		"success case no workers": {
			prepareMock: prepareMockOkNoWorkers,
			wantCharts:  len(charts),
			wantMetrics: map[string]int64{
				"workers_exceptions": 0,
				"workers_harakiris":  0,
				"workers_requests":   0,
				"workers_respawns":   0,
				"workers_tx":         0,
			},
			disconnectBeforeCleanup: true,
			disconnectAfterCleanup:  true,
		},
		"unexpected response": {
			prepareMock:             prepareMockUnexpectedResponse,
			wantCharts:              len(charts),
			disconnectBeforeCleanup: true,
			disconnectAfterCleanup:  true,
		},
		"empty response": {
			prepareMock:             prepareMockEmptyResponse,
			wantCharts:              len(charts),
			disconnectBeforeCleanup: true,
			disconnectAfterCleanup:  true,
		},
		"err on connect": {
			prepareMock:             prepareMockErrOnConnect,
			wantCharts:              len(charts),
			disconnectBeforeCleanup: false,
			disconnectAfterCleanup:  false,
		},
		"err on query stats": {
			prepareMock:             prepareMockErrOnQueryStats,
			wantCharts:              len(charts),
			disconnectBeforeCleanup: true,
			disconnectAfterCleanup:  true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			uw := New()
			mock := test.prepareMock()
			uw.newConn = func(config Config) uwsgiConn { return mock }

			mx := uw.Collect()

			require.Equal(t, test.wantMetrics, mx)

			if len(test.wantMetrics) > 0 {
				module.TestMetricsHasAllChartsDims(t, uw.Charts(), mx)
			}
			assert.Equal(t, test.wantCharts, len(*uw.Charts()), "want charts")

			assert.Equal(t, test.disconnectBeforeCleanup, mock.disconnectCalled, "disconnect before cleanup")
			uw.Cleanup()
			assert.Equal(t, test.disconnectAfterCleanup, mock.disconnectCalled, "disconnect after cleanup")
		})
	}
}

func prepareMockOk() *mockUwsgiConn {
	return &mockUwsgiConn{
		statsResponse: dataStats,
	}
}

func prepareMockOkNoWorkers() *mockUwsgiConn {
	return &mockUwsgiConn{
		statsResponse: dataStatsNoWorkers,
	}
}

func prepareMockErrOnConnect() *mockUwsgiConn {
	return &mockUwsgiConn{
		errOnConnect: true,
	}
}

func prepareMockErrOnQueryStats() *mockUwsgiConn {
	return &mockUwsgiConn{
		errOnQueryStats: true,
	}
}

func prepareMockUnexpectedResponse() *mockUwsgiConn {
	return &mockUwsgiConn{
		statsResponse: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
	}
}

func prepareMockEmptyResponse() *mockUwsgiConn {
	return &mockUwsgiConn{}
}

type mockUwsgiConn struct {
	errOnConnect     bool
	errOnQueryStats  bool
	statsResponse    []byte
	disconnectCalled bool
}

func (m *mockUwsgiConn) connect() error {
	if m.errOnConnect {
		return errors.New("mock.connect() error")
	}
	return nil
}

func (m *mockUwsgiConn) disconnect() {
	m.disconnectCalled = true
}

func (m *mockUwsgiConn) queryStats() ([]byte, error) {
	if m.errOnQueryStats {
		return nil, errors.New("mock.queryStats() error")
	}
	return m.statsResponse, nil
}
