// Copyright 2020 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsm

import (
	"fmt"
	"strings"

	"github.com/nats-io/jsm.go/api"
)

type StreamTemplate struct {
	cfg     StreamTemplateConfig
	streams []string
}

type StreamTemplateConfig struct {
	api.StreamTemplateConfig

	conn  *reqoptions
	ropts []RequestOption
}

// NewStreamTemplate creates a new template
func NewStreamTemplate(name string, maxStreams uint32, config api.StreamConfig, opts ...StreamOption) (template *StreamTemplate, err error) {
	cfg, err := NewStreamConfiguration(config, opts...)
	if err != nil {
		return nil, err
	}

	tc := api.StreamTemplateConfig{
		Name:       name,
		Config:     &cfg.StreamConfig,
		MaxStreams: maxStreams,
	}

	valid, errs := tc.Validate()
	if !valid {
		return nil, fmt.Errorf("configuration validation failed: %s", strings.Join(errs, ", "))
	}

	var resp api.JSApiStreamTemplateCreateResponse
	err = jsonRequest(fmt.Sprintf(api.JSApiTemplateCreateT, name), tc, &resp, cfg.conn)
	if err != nil {
		return nil, err
	}

	// TODO speed up with the info in resp
	return LoadStreamTemplate(name, cfg.ropts...)
}

// LoadOrNewStreamTemplate loads an existing template, else creates a new one based on config
func LoadOrNewStreamTemplate(name string, maxStreams uint32, config api.StreamConfig, opts ...StreamOption) (template *StreamTemplate, err error) {
	template, err = LoadStreamTemplate(name)
	if template != nil && err == nil {
		return template, nil
	}

	return NewStreamTemplate(name, maxStreams, config, opts...)
}

// LoadStreamTemplate loads a given stream template from JetStream
func LoadStreamTemplate(name string, opts ...RequestOption) (template *StreamTemplate, err error) {
	if !IsValidName(name) {
		return nil, fmt.Errorf("%q is not a valid stream template name", name)
	}

	conn, err := newreqoptions(opts...)
	if err != nil {
		return nil, err
	}

	template = &StreamTemplate{
		cfg: StreamTemplateConfig{
			StreamTemplateConfig: api.StreamTemplateConfig{Name: name},
			conn:                 conn,
			ropts:                opts,
		},
	}

	err = loadConfigForStreamTemplate(template)
	if err != nil {
		return nil, err
	}

	return template, nil
}

func loadConfigForStreamTemplate(template *StreamTemplate) (err error) {
	var resp api.JSApiStreamTemplateInfoResponse
	err = jsonRequest(fmt.Sprintf(api.JSApiTemplateInfoT, template.Name()), nil, &resp, template.cfg.conn)
	if err != nil {
		return err
	}

	template.cfg.StreamTemplateConfig = *resp.Config
	template.streams = resp.Streams

	return nil
}

// Delete deletes the StreamTemplate, after this the StreamTemplate object should be disposed
func (t *StreamTemplate) Delete() error {
	var resp api.JSApiStreamTemplateDeleteResponse
	err := jsonRequest(fmt.Sprintf(api.JSApiTemplateDeleteT, t.Name()), nil, &resp, t.cfg.conn)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("unknown error while deleting %s", t.Name())
	}

	return nil
}

// Reset reloads the Stream Template configuration and state from the JetStream server
func (t *StreamTemplate) Reset() error {
	return loadConfigForStreamTemplate(t)
}

func (t *StreamTemplate) Configuration() api.StreamTemplateConfig {
	return t.cfg.StreamTemplateConfig
}

func (t *StreamTemplate) StreamConfiguration() api.StreamConfig {
	return *t.cfg.Config
}

func (t *StreamTemplate) Name() string {
	return t.cfg.Name
}

func (t *StreamTemplate) MaxStreams() uint32 {
	return t.cfg.MaxStreams
}

func (t *StreamTemplate) Streams() []string {
	return t.streams
}
