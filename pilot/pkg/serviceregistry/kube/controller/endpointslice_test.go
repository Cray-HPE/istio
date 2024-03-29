// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"reflect"
	"testing"

	"istio.io/api/label"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config/host"
)

func TestGetLocalityFromTopology(t *testing.T) {
	cases := []struct {
		name     string
		topology map[string]string
		locality string
	}{
		{
			"all standard kubernetes labels",
			map[string]string{
				NodeRegionLabelGA: "region",
				NodeZoneLabelGA:   "zone",
			},
			"region/zone",
		},
		{
			"all standard kubernetes labels and Istio custom labels",
			map[string]string{
				NodeRegionLabelGA:          "region",
				NodeZoneLabelGA:            "zone",
				label.TopologySubzone.Name: "subzone",
			},
			"region/zone/subzone",
		},
		{
			"missing zone",
			map[string]string{
				NodeRegionLabelGA: "region",
			},
			"region",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := getLocalityFromTopology(tt.topology)
			if !reflect.DeepEqual(tt.locality, got) {
				t.Fatalf("Expected %v, got %v", tt.topology, got)
			}
		})
	}
}

func TestEndpointSliceCache(t *testing.T) {
	cache := newEndpointSliceCache()
	hostname := host.Name("foo")

	// add a endpoint
	ep1 := &model.IstioEndpoint{
		Address:         "1.2.3.4",
		ServicePortName: "http",
	}
	cache.Update(hostname, "slice1", []*model.IstioEndpoint{ep1})
	if !testEndpointsEqual(cache.Get(hostname), []*model.IstioEndpoint{ep1}) {
		t.Fatalf("unexpected endpoints")
	}

	// add a new endpoint
	ep2 := &model.IstioEndpoint{
		Address:         "2.3.4.5",
		ServicePortName: "http",
	}
	cache.Update(hostname, "slice1", []*model.IstioEndpoint{ep1, ep2})
	if !testEndpointsEqual(cache.Get(hostname), []*model.IstioEndpoint{ep1, ep2}) {
		t.Fatalf("unexpected endpoints")
	}

	// change service port name
	ep1 = &model.IstioEndpoint{
		Address:         "1.2.3.4",
		ServicePortName: "http2",
	}
	ep2 = &model.IstioEndpoint{
		Address:         "2.3.4.5",
		ServicePortName: "http2",
	}
	cache.Update(hostname, "slice1", []*model.IstioEndpoint{ep1, ep2})
	if !testEndpointsEqual(cache.Get(hostname), []*model.IstioEndpoint{ep1, ep2}) {
		t.Fatalf("unexpected endpoints")
	}

	// add a new slice
	ep3 := &model.IstioEndpoint{
		Address:         "3.4.5.6",
		ServicePortName: "http2",
	}
	cache.Update(hostname, "slice2", []*model.IstioEndpoint{ep3})
	if !testEndpointsEqual(cache.Get(hostname), []*model.IstioEndpoint{ep1, ep2, ep3}) {
		t.Fatalf("unexpected endpoints")
	}

	// dedup when transitioning
	cache.Update(hostname, "slice2", []*model.IstioEndpoint{ep2, ep3})
	if !testEndpointsEqual(cache.Get(hostname), []*model.IstioEndpoint{ep1, ep2, ep3}) {
		t.Fatalf("unexpected endpoints")
	}

	cache.Delete(hostname, "slice1")
	if !testEndpointsEqual(cache.Get(hostname), []*model.IstioEndpoint{ep2, ep3}) {
		t.Fatalf("unexpected endpoints")
	}

	cache.Delete(hostname, "slice2")
	if cache.Get(hostname) != nil {
		t.Fatalf("unexpected endpoints")
	}
}

func testEndpointsEqual(a, b []*model.IstioEndpoint) bool {
	if len(a) != len(b) {
		return false
	}
	m1 := make(map[endpointKey]int)
	m2 := make(map[endpointKey]int)
	for _, i := range a {
		m1[endpointKey{i.Address, i.ServicePortName}]++
	}
	for _, i := range b {
		m2[endpointKey{i.Address, i.ServicePortName}]++
	}
	return reflect.DeepEqual(m1, m2)
}
