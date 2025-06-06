// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package xml for config provider.
//
// depend on github.com/beego/x2j.
//
// go install github.com/beego/x2j.
//
// Usage:
//
//	import(
//	  _ "github.com/beego/beego/v2/core/config/xml"
//	    "github.com/beego/beego/v2/core/config"
//	)
//
//	cnf, err := config.NewConfig("xml", "config.xml")
package xml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/mitchellh/mapstructure"

	"github.com/beego/x2j"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

// Config is a xml config parser and implements Config interface.
// xml configurations should be included in <config></config> tag.
// only support key/value pair as <key>value</key> as each item.
type Config struct{}

// Parse returns a ConfigContainer with parsed xml config map.
func (xc *Config) Parse(filename string) (config.Configer, error) {
	context, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return xc.ParseData(context)
}

// ParseData xml data
func (xc *Config) ParseData(data []byte) (config.Configer, error) {
	x := &ConfigContainer{data: make(map[string]interface{})}

	d, err := x2j.DocToMap(string(data))
	if err != nil {
		return nil, err
	}

	v := d["config"]
	if v == nil {
		return nil, fmt.Errorf("xml parse should include in <config></config> tags")
	}

	confVal, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("xml parse <config></config> tags should include sub tags")
	}

	x.data = config.ExpandValueEnvForMap(confVal)

	return x, nil
}

// ConfigContainer is a Config which represents the xml configuration.
type ConfigContainer struct {
	data map[string]interface{}
	sync.Mutex
}

// Unmarshaler is a little be inconvenient since the xml library doesn't know type.
// So when you use
// <id>1</id>
// The "1" is a string, not int
func (c *ConfigContainer) Unmarshaler(prefix string, obj interface{}, opt ...config.DecodeOption) error {
	sub, err := c.sub(prefix)
	if err != nil {
		return err
	}
	return mapstructure.Decode(sub, obj)
}

func (c *ConfigContainer) Sub(key string) (config.Configer, error) {
	sub, err := c.sub(key)
	if err != nil {
		return nil, err
	}

	return &ConfigContainer{
		data: sub,
	}, nil
}

func (c *ConfigContainer) sub(key string) (map[string]interface{}, error) {
	if key == "" {
		return c.data, nil
	}
	value, ok := c.data[key]
	if !ok {
		return nil, fmt.Errorf("the key is not found: %s", key)
	}
	res, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("the value of this key is not a structure: %s", key)
	}
	return res, nil
}

func (c *ConfigContainer) OnChange(key string, fn func(value string)) {
	logs.Warn("Unsupported operation")
}

// Bool returns the boolean value for a given key.
func (c *ConfigContainer) Bool(key string) (bool, error) {
	if v := c.data[key]; v != nil {
		return config.ParseBool(v)
	}
	return false, fmt.Errorf("not exist key: %q", key)
}

// DefaultBool return the bool value if has no error
// otherwise return the defaultVal
func (c *ConfigContainer) DefaultBool(key string, defaultVal bool) bool {
	v, err := c.Bool(key)
	if err != nil {
		return defaultVal
	}
	return v
}

// Int returns the integer value for a given key.
func (c *ConfigContainer) Int(key string) (int, error) {
	return strconv.Atoi(c.data[key].(string))
}

// DefaultInt returns the integer value for a given key.
// if err != nil return defaultVal
func (c *ConfigContainer) DefaultInt(key string, defaultVal int) int {
	v, err := c.Int(key)
	if err != nil {
		return defaultVal
	}
	return v
}

// Int64 returns the int64 value for a given key.
func (c *ConfigContainer) Int64(key string) (int64, error) {
	return strconv.ParseInt(c.data[key].(string), 10, 64)
}

// DefaultInt64 returns the int64 value for a given key.
// if err != nil return defaultVal
func (c *ConfigContainer) DefaultInt64(key string, defaultVal int64) int64 {
	v, err := c.Int64(key)
	if err != nil {
		return defaultVal
	}
	return v
}

// Float returns the float value for a given key.
func (c *ConfigContainer) Float(key string) (float64, error) {
	return strconv.ParseFloat(c.data[key].(string), 64)
}

// DefaultFloat returns the float64 value for a given key.
// if err != nil return defaultVal
func (c *ConfigContainer) DefaultFloat(key string, defaultVal float64) float64 {
	v, err := c.Float(key)
	if err != nil {
		return defaultVal
	}
	return v
}

// String returns the string value for a given key.
func (c *ConfigContainer) String(key string) (string, error) {
	if v, ok := c.data[key].(string); ok {
		return v, nil
	}
	return "", nil
}

// DefaultString returns the string value for a given key.
// if err != nil return defaultVal
func (c *ConfigContainer) DefaultString(key string, defaultVal string) string {
	v, err := c.String(key)
	if v == "" || err != nil {
		return defaultVal
	}
	return v
}

// Strings returns the []string value for a given key.
func (c *ConfigContainer) Strings(key string) ([]string, error) {
	v, err := c.String(key)
	if v == "" || err != nil {
		return nil, err
	}
	return strings.Split(v, ";"), nil
}

// DefaultStrings returns the []string value for a given key.
// if err != nil return defaultVal
func (c *ConfigContainer) DefaultStrings(key string, defaultVal []string) []string {
	v, err := c.Strings(key)
	if v == nil || err != nil {
		return defaultVal
	}
	return v
}

// GetSection returns map for the given section
func (c *ConfigContainer) GetSection(section string) (map[string]string, error) {
	if v, ok := c.data[section].(map[string]interface{}); ok {
		mapstr := make(map[string]string)
		for k, val := range v {
			mapstr[k] = config.ToString(val)
		}
		return mapstr, nil
	}
	return nil, fmt.Errorf("section '%s' not found", section)
}

// SaveConfigFile save the config into file
func (c *ConfigContainer) SaveConfigFile(filename string) (err error) {
	// Write configuration file by filename.
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := xml.MarshalIndent(c.data, "  ", "    ")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

// Set writes a new value for key.
func (c *ConfigContainer) Set(key, val string) error {
	c.Lock()
	defer c.Unlock()
	c.data[key] = val
	return nil
}

// DIY returns the raw value by a given key.
func (c *ConfigContainer) DIY(key string) (v interface{}, err error) {
	if v, ok := c.data[key]; ok {
		return v, nil
	}
	return nil, errors.New("not exist key")
}

func init() {
	config.Register("xml", &Config{})
}
