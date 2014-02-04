/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package types

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"strings"
)

type GenericContainer interface {
	GetApp() string
	GetSha() string
	GetID() string
	SetDockerID(string)
	GetDockerID() string
	GetDockerRepo() string
	GetIP() string
	SetIP(string)
	GetSSHPort() uint16
}

type Container struct {
	ID             string
	DockerID       string
	IP             string
	Host           string
	PrimaryPort    uint16
	SecondaryPorts []uint16
	SSHPort        uint16
	App            string
	Sha            string
	Env            string
	Manifest       *Manifest
}

func (c *Container) GetID() string {
	return c.ID
}

func (c *Container) GetApp() string {
	return c.App
}

func (c *Container) GetSha() string {
	return c.Sha
}

func (c *Container) SetDockerID(id string) {
	c.DockerID = id
}

func (c *Container) GetDockerID() string {
	return c.DockerID
}

func (c *Container) GetDockerRepo() string {
	return "apps"
}

func (c *Container) SetIP(ip string) {
	c.IP = ip
}

func (c *Container) GetIP() string {
	return c.IP
}

func (c *Container) GetSSHPort() uint16 {
	return c.SSHPort
}

func (c *Container) RandomID() string {
	return c.ID[strings.LastIndex(c.ID, "-")+1:]
}

func (c *Container) String() string {
	return fmt.Sprintf(`%s
IP              : %s
Host            : %s
Primary Port    : %d
SSH Port        : %d
Secondary Ports : %v
App             : %s
SHA             : %s
CPU Shares      : %d
Memory Limit    : %d
Docker ID       : %s`, c.ID, c.IP, c.Host, c.PrimaryPort, c.SSHPort, c.SecondaryPorts, c.App, c.Sha,
		c.Manifest.CPUShares, c.Manifest.MemoryLimit, c.DockerID)
}

// NOTE[jigish]: ONLY for TOML parsing
type ManifestTOML struct {
	Name        string
	Description string
	Instances   uint
	CPUShares   uint `toml:"cpu_shares"`   // should be 1 or any multiple of 5
	MemoryLimit uint `toml:"memory_limit"` // should be a multiple of 256 (MBytes)
	Image       string
	AppType     string      `toml:"app_type"`
	RunCommand  interface{} `toml:"run_command"` // can be string or array
	DepNames    []string    `toml:"dependencies"`
}

type DepsType map[string]*AppDep
type AppDep struct {
	SecurityGroup []string
	DataMap       map[string]interface{}
	EncryptedData string
}

type Manifest struct {
	Name        string
	Description string
	Instances   uint
	CPUShares   uint
	MemoryLimit uint
	Image       string
	AppType     string
	RunCommands []string
	Deps        DepsType
}

func (m *Manifest) Dup() *Manifest {
	runCommands := make([]string, len(m.RunCommands))
	for i, cmd := range m.RunCommands {
		runCommands[i] = cmd
	}
	deps := DepsType{}
	for key, val := range m.Deps {
		deps[key] = &AppDep{
			SecurityGroup: make([]string, len(val.SecurityGroup)),
			DataMap:       map[string]interface{}{},
		}
		for i, ipAndPort := range val.SecurityGroup {
			deps[key].SecurityGroup[i] = ipAndPort
		}
		for innerKey, innerVal := range val.DataMap {
			deps[key].DataMap[innerKey] = innerVal
		}
		deps[key].EncryptedData = val.EncryptedData
	}
	return &Manifest{
		Name:        m.Name,
		Description: m.Description,
		Instances:   m.Instances,
		CPUShares:   m.CPUShares,
		MemoryLimit: m.MemoryLimit,
		Image:       m.Image,
		AppType:     m.AppType,
		RunCommands: runCommands,
		Deps:        deps,
	}
}

func CreateManifest(mt *ManifestTOML) (*Manifest, error) {
	deps := DepsType{}
	for _, name := range mt.DepNames {
		deps[name] = &AppDep{} // set it here so we can check for it in DepNames()
	}
	var cmds []string
	switch runCommand := mt.RunCommand.(type) {
	case string:
		cmds = []string{runCommand}
	case []interface{}:
		cmds = make([]string, 1, 1)
		for _, cmd := range runCommand {
			cmdStr, ok := cmd.(string)
			if ok {
				cmds = append(cmds, cmdStr)
			} else {
				return nil, errors.New("Invalid Manifest: non-string element in run_command array!")
			}
		}
	default:
		return nil, errors.New("Invalid Manifest: run_command should be string or []string")
	}
	return &Manifest{
		Name:        mt.Name,
		Description: mt.Description,
		Instances:   mt.Instances,
		CPUShares:   mt.CPUShares,
		MemoryLimit: mt.MemoryLimit,
		Image:       mt.Image,
		AppType:     mt.AppType,
		RunCommands: cmds,
		Deps:        deps,
	}, nil
}

func (m *Manifest) DepNames() []string {
	names := make([]string, len(m.Deps))
	i := 0
	for name, _ := range m.Deps {
		names[i] = name
		i++
	}
	return names
}

func ReadManifest(r io.Reader) (*Manifest, error) {
	var manifestTOML ManifestTOML
	_, err := toml.DecodeReader(r, &manifestTOML)
	if err != nil {
		return nil, errors.New("Parse Manifest Error: " + err.Error())
	}
	return CreateManifest(&manifestTOML)
}

// ----------------------------------------------------------------------------------------------------------
// Supervisor RPC Types
// ----------------------------------------------------------------------------------------------------------

// ------------ Health Check ------------
// Used to check the health and stats of Supervisor
type SupervisorHealthCheckArg struct {
}

type ResourceStats struct {
	Total uint
	Used  uint
	Free  uint
}

type SupervisorHealthCheckReply struct {
	Containers *ResourceStats
	CPUShares  *ResourceStats
	Memory     *ResourceStats
	Region     string
	Zone       string
	Status     string
}

// ------------ Deploy ------------
// Used to deploy a new app/sha
type SupervisorDeployArg struct {
	Host        string
	App         string
	Sha         string
	Env         string
	ContainerID string
	Manifest    *Manifest
}

type SupervisorDeployReply struct {
	Status    string
	Container *Container
}

// ------------ Teardown ------------
// Used to teardown a container
type SupervisorTeardownArg struct {
	ContainerIDs []string
	All          bool
}

type SupervisorTeardownReply struct {
	ContainerIDs []string
	Status       string
}

// ------------ Get ------------
// Used to get a container
type SupervisorGetArg struct {
	ContainerID string
}

type SupervisorGetReply struct {
	Container *Container
	Status    string
}

// ------------ List ------------
// List Supervisor Containers
type SupervisorListArg struct {
}

type SupervisorListReply struct {
	Containers  map[string]*Container
	UnusedPorts []uint16
}

// ------------ Authorize SSH ------------
// Authorize SSH
type SupervisorAuthorizeSSHArg struct {
	ContainerID string
	User        string
	PublicKey   string
}

type SupervisorAuthorizeSSHReply struct {
	Port   uint16
	Status string
}

// ------------ Deauthorize SSH ------------
// Deauthorize SSH
type SupervisorDeauthorizeSSHArg struct {
	ContainerID string
	User        string
}

type SupervisorDeauthorizeSSHReply struct {
	Status string
}

// ------------ Container Maintenance ------------
// Set Container Maintenance Mode
type SupervisorContainerMaintenanceArg struct {
	ContainerID string
	Maintenance bool
}

type SupervisorContainerMaintenanceReply struct {
	Status string
}

// ------------ Idle ------------
// Check if Idle
type SupervisorIdleArg struct {
}

type SupervisorIdleReply struct {
	Idle   bool
	Status string
}
