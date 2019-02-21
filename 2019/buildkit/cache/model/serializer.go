package model

import (
	"bytes"
	"fmt"
	"strings"
)

func (s *Service) translatePorts() {
	for _, c := range s.Containers {
		if c == nil {
			// TODO: catch this earlier
			continue
		}

		ports := []string{}

		if c.Ports != nil {
			for _, p := range c.Ports {
				parts := strings.SplitN(p, ":", 4)
				if len(parts) == 4 {
					ports = append(ports, parts[3])
				} else {
					ports = append(ports, p)
				}
			}
		}

		c.Ports = ports
		if c.Ingress != nil {
			for _, i := range c.Ingress {
				if i.Host == "" {
					i.Host = s.Name
				}
				if i.Path == "" {
					i.Path = "/"
				}
			}
		}
	}
}

func (s *Service) translateVolumes() {
	for vName, v := range s.Volumes {
		if v == nil {
			v = &Volume{}
			s.Volumes[vName] = v
		}
		v.Name = vName
		if v.Persistent && v.Size == "" {
			v.Size = "20Gi"
		}
	}
}

//MarshalYAML serializes e into a YAML document. The return value is a string; It will fail if e has an empty name.
func (e *EnvVar) MarshalYAML() (interface{}, error) {
	if e.Name == "" {
		return "", fmt.Errorf("missing values")
	}

	var buffer bytes.Buffer
	buffer.WriteString(e.Name)
	buffer.WriteString("=")
	if e.Value != "" {
		buffer.WriteString(e.Value)
	}

	return buffer.String(), nil
}

//UnmarshalYAML parses the yaml element and sets the values of e; it will return an error if the parsing fails, or
//if the format is incorrect
func (e *EnvVar) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var envvar string
	if err := unmarshal(&envvar); err != nil {
		return err
	}

	envvar = strings.TrimPrefix(envvar, "=")

	parts := strings.SplitN(envvar, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("Invalid environment variable syntax")
	}

	e.Name = parts[0]
	e.Value = parts[1]
	return nil
}
