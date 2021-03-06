/***
Copyright 2014 Cisco Systems Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"encoding/json"
	"fmt"

	"github.com/contiv/netplugin/core"
	"github.com/contiv/netplugin/drivers"
	"github.com/contiv/netplugin/netmaster/mastercfg"
	"github.com/jainvipin/bitset"
)

const (
	// AutoVXLANResource is a string description of the type of resource.
	AutoVXLANResource = "auto-vxlan"
)

const (
	vXLANResourceConfigPathPrefix = mastercfg.StateConfigPath + AutoVXLANResource + "/"
	vXLANResourceConfigPath       = vXLANResourceConfigPathPrefix + "%s"
	vXLANResourceOperPathPrefix   = drivers.StateOperPath + AutoVXLANResource + "/"
	vXLANResourceOperPath         = vXLANResourceOperPathPrefix + "%s"
)

// AutoVXLANCfgResource implements the Resource interface for an 'auto-vxlan' resource.
// 'auto-vxlan' resource allocates a vxlan from a range of vxlan encaps specified
// at time of resource instantiation
type AutoVXLANCfgResource struct {
	core.CommonState
	VXLANs     *bitset.BitSet `json:"vxlans"`
	LocalVLANs *bitset.BitSet `json:"LocalVLANs"`
}

// VXLANVLANPair Pairs a VXLAN tag with a VLAN tag.
type VXLANVLANPair struct {
	VXLAN uint
	VLAN  uint
}

// Write the state.
func (r *AutoVXLANCfgResource) Write() error {
	key := fmt.Sprintf(vXLANResourceConfigPath, r.ID)
	return r.StateDriver.WriteState(key, r, json.Marshal)
}

// Read the state.
func (r *AutoVXLANCfgResource) Read(id string) error {
	key := fmt.Sprintf(vXLANResourceConfigPath, id)
	return r.StateDriver.ReadState(key, r, json.Unmarshal)
}

// Clear the state.
func (r *AutoVXLANCfgResource) Clear() error {
	key := fmt.Sprintf(vXLANResourceConfigPath, r.ID)
	return r.StateDriver.ClearState(key)
}

// ReadAll reads all the state from the resource.
func (r *AutoVXLANCfgResource) ReadAll() ([]core.State, error) {
	return r.StateDriver.ReadAllState(vXLANResourceConfigPathPrefix, r,
		json.Unmarshal)
}

// Init the resource.
func (r *AutoVXLANCfgResource) Init(rsrcCfg interface{}) error {
	cfg, ok := rsrcCfg.(*AutoVXLANCfgResource)
	if !ok {
		return core.Errorf("Invalid vxlan resource config.")
	}
	r.VXLANs = cfg.VXLANs
	r.LocalVLANs = cfg.LocalVLANs
	err := r.Write()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			r.Clear()
		}
	}()

	oper := &AutoVXLANOperResource{FreeVXLANs: r.VXLANs, FreeLocalVLANs: r.LocalVLANs}
	oper.StateDriver = r.StateDriver
	oper.ID = r.ID
	err = oper.Write()
	if err != nil {
		return err
	}

	return nil
}

// Deinit the resource.
func (r *AutoVXLANCfgResource) Deinit() {
	oper := &AutoVXLANOperResource{}
	oper.StateDriver = r.StateDriver
	err := oper.Read(r.ID)
	if err != nil {
		// continue cleanup
	} else {
		err = oper.Clear()
		if err != nil {
			// continue cleanup
		}
	}

	r.Clear()
}

// Description is a string description of the resource. Returns AutoVXLANResource.
func (r *AutoVXLANCfgResource) Description() string {
	return AutoVXLANResource
}

// Allocate allocates a new resource.
func (r *AutoVXLANCfgResource) Allocate() (interface{}, error) {
	oper := &AutoVXLANOperResource{}
	oper.StateDriver = r.StateDriver
	err := oper.Read(r.ID)
	if err != nil {
		return nil, err
	}

	vxlan, ok := oper.FreeVXLANs.NextSet(0)
	if !ok {
		return nil, core.Errorf("no vxlans available.")
	}

	vlan, ok := oper.FreeLocalVLANs.NextSet(0)
	if !ok {
		return nil, core.Errorf("no local vlans available.")
	}

	oper.FreeVXLANs.Clear(vxlan)
	oper.FreeLocalVLANs.Clear(vlan)

	err = oper.Write()
	if err != nil {
		return nil, err
	}
	return VXLANVLANPair{VXLAN: vxlan, VLAN: vlan}, nil
}

// Deallocate removes and cleans up a resource.
func (r *AutoVXLANCfgResource) Deallocate(value interface{}) error {
	oper := &AutoVXLANOperResource{}
	oper.StateDriver = r.StateDriver
	err := oper.Read(r.ID)
	if err != nil {
		return err
	}

	pair, ok := value.(VXLANVLANPair)
	if !ok {
		return core.Errorf("Invalid type for vxlan-vlan pair")
	}
	vxlan := pair.VXLAN
	oper.FreeVXLANs.Set(vxlan)
	vlan := pair.VLAN
	oper.FreeLocalVLANs.Set(vlan)

	err = oper.Write()
	if err != nil {
		return err
	}
	return nil
}

// AutoVXLANOperResource is an implementation of core.State
type AutoVXLANOperResource struct {
	core.CommonState
	FreeVXLANs     *bitset.BitSet `json:"freeVXLANs"`
	FreeLocalVLANs *bitset.BitSet `json:"freeLocalVLANs"`
}

// Write the state.
func (r *AutoVXLANOperResource) Write() error {
	key := fmt.Sprintf(vXLANResourceOperPath, r.ID)
	return r.StateDriver.WriteState(key, r, json.Marshal)
}

// Read the state.
func (r *AutoVXLANOperResource) Read(id string) error {
	key := fmt.Sprintf(vXLANResourceOperPath, id)
	return r.StateDriver.ReadState(key, r, json.Unmarshal)
}

// ReadAll the state for the given type.
func (r *AutoVXLANOperResource) ReadAll() ([]core.State, error) {
	return r.StateDriver.ReadAllState(vXLANResourceOperPathPrefix, r,
		json.Unmarshal)
}

// Clear the state.
func (r *AutoVXLANOperResource) Clear() error {
	key := fmt.Sprintf(vXLANResourceOperPath, r.ID)
	return r.StateDriver.ClearState(key)
}
