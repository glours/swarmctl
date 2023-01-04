package config

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/moby/swarmctl/internal/test"
	. "github.com/moby/swarmctl/internal/test/builders" // Import builders to get the builder function as package function
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestConfigInspectErrors(t *testing.T) {
	testCases := []struct {
		args              []string
		flags             map[string]string
		configInspectFunc func(configID string) (swarm.Config, []byte, error)
		expectedError     string
	}{
		{
			expectedError: "requires at least 1 argument",
		},
		{
			args: []string{"foo"},
			configInspectFunc: func(configID string) (swarm.Config, []byte, error) {
				return swarm.Config{}, nil, errors.Errorf("error while inspecting the config")
			},
			expectedError: "error while inspecting the config",
		},
		{
			args: []string{"foo"},
			flags: map[string]string{
				"format": "{{invalid format}}",
			},
			expectedError: "template parsing error",
		},
		{
			args: []string{"foo", "bar"},
			configInspectFunc: func(configID string) (swarm.Config, []byte, error) {
				if configID == "foo" {
					return *Config(ConfigName("foo")), nil, nil // nolint: typecheck
				}
				return swarm.Config{}, nil, errors.Errorf("error while inspecting the config")
			},
			expectedError: "error while inspecting the config",
		},
	}
	for _, tc := range testCases {
		cmd := newConfigInspectCommand(
			test.NewFakeCli(&fakeClient{
				configInspectFunc: tc.configInspectFunc,
			}),
		)
		cmd.SetArgs(tc.args)
		for key, value := range tc.flags {
			cmd.Flags().Set(key, value) //nolint:errcheck
		}
		cmd.SetOut(io.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestConfigInspectWithoutFormat(t *testing.T) {
	testCases := []struct {
		name              string
		args              []string
		configInspectFunc func(configID string) (swarm.Config, []byte, error)
	}{
		{
			name: "single-config",
			args: []string{"foo"},
			configInspectFunc: func(name string) (swarm.Config, []byte, error) {
				if name != "foo" {
					return swarm.Config{}, nil, errors.Errorf("Invalid name, expected %s, got %s", "foo", name)
				}
				return *Config(ConfigID("ID-foo"), ConfigName("foo")), nil, nil // nolint: typecheck
			},
		},
		{
			name: "multiple-configs-with-labels",
			args: []string{"foo", "bar"},
			configInspectFunc: func(name string) (swarm.Config, []byte, error) {
				return *Config(ConfigID("ID-"+name), ConfigName(name), ConfigLabels(map[string]string{ // nolint: typecheck
					"label1": "label-foo",
				})), nil, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{configInspectFunc: tc.configInspectFunc})
		cmd := newConfigInspectCommand(cli)
		cmd.SetArgs(tc.args)
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("config-inspect-without-format.%s.golden", tc.name))
	}
}

func TestConfigInspectWithFormat(t *testing.T) {
	configInspectFunc := func(name string) (swarm.Config, []byte, error) {
		return *Config(ConfigName("foo"), ConfigLabels(map[string]string{ // nolint: typecheck
			"label1": "label-foo",
		})), nil, nil
	}
	testCases := []struct {
		name              string
		format            string
		args              []string
		configInspectFunc func(name string) (swarm.Config, []byte, error)
	}{
		{
			name:              "simple-template",
			format:            "{{.Spec.Name}}",
			args:              []string{"foo"},
			configInspectFunc: configInspectFunc,
		},
		{
			name:              "json-template",
			format:            "{{json .Spec.Labels}}",
			args:              []string{"foo"},
			configInspectFunc: configInspectFunc,
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			configInspectFunc: tc.configInspectFunc,
		})
		cmd := newConfigInspectCommand(cli)
		cmd.SetArgs(tc.args)
		cmd.Flags().Set("format", tc.format) //nolint:errcheck
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("config-inspect-with-format.%s.golden", tc.name))
	}
}

func TestConfigInspectPretty(t *testing.T) {
	testCases := []struct {
		name              string
		configInspectFunc func(string) (swarm.Config, []byte, error)
	}{
		{
			name: "simple",
			configInspectFunc: func(id string) (swarm.Config, []byte, error) {
				return *Config( // nolint: typecheck
					ConfigLabels(map[string]string{ // nolint: typecheck
						"lbl1": "value1",
					}),
					ConfigID("configID"),               // nolint: typecheck
					ConfigName("configName"),           // nolint: typecheck
					ConfigCreatedAt(time.Time{}),       // nolint: typecheck
					ConfigUpdatedAt(time.Time{}),       // nolint: typecheck
					ConfigData([]byte("payload here")), // nolint: typecheck
				), []byte{}, nil
			},
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			configInspectFunc: tc.configInspectFunc,
		})
		cmd := newConfigInspectCommand(cli)

		cmd.SetArgs([]string{"configID"})
		cmd.Flags().Set("pretty", "true") //nolint:errcheck
		assert.NilError(t, cmd.Execute())
		golden.Assert(t, cli.OutBuffer().String(), fmt.Sprintf("config-inspect-pretty.%s.golden", tc.name))
	}
}
