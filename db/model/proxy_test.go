/*
 * Copyright 2018-present Open Networking Foundation

 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

 * http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package model

import (
	"encoding/hex"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/opencord/voltha-go/protos/common"
	"github.com/opencord/voltha-go/protos/openflow_13"
	"github.com/opencord/voltha-go/protos/voltha"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

var (
	TestProxy_Root                  *root
	TestProxy_Root_LogicalDevice    *Proxy
	TestProxy_Root_Device           *Proxy
	TestProxy_DeviceId              string
	TestProxy_LogicalDeviceId       string
	TestProxy_TargetDeviceId        string
	TestProxy_TargetLogicalDeviceId string
	TestProxy_LogicalPorts          []*voltha.LogicalPort
	TestProxy_Ports                 []*voltha.Port
	TestProxy_Stats                 *openflow_13.OfpFlowStats
	TestProxy_Flows                 *openflow_13.Flows
	TestProxy_Device                *voltha.Device
	TestProxy_LogicalDevice         *voltha.LogicalDevice
)

func init() {
	//log.AddPackage(log.JSON, log.InfoLevel, log.Fields{"instanceId": "DB_MODEL"})
	//log.UpdateAllLoggers(log.Fields{"instanceId": "PROXY_LOAD_TEST"})
	TestProxy_Root = NewRoot(&voltha.Voltha{}, nil)
	TestProxy_Root_LogicalDevice = TestProxy_Root.CreateProxy("/", false)
	TestProxy_Root_Device = TestProxy_Root.CreateProxy("/", false)

	TestProxy_LogicalPorts = []*voltha.LogicalPort{
		{
			Id:           "123",
			DeviceId:     "logicalport-0-device-id",
			DevicePortNo: 123,
			RootPort:     false,
		},
	}
	TestProxy_Ports = []*voltha.Port{
		{
			PortNo:     123,
			Label:      "test-port-0",
			Type:       voltha.Port_PON_OLT,
			AdminState: common.AdminState_ENABLED,
			OperStatus: common.OperStatus_ACTIVE,
			DeviceId:   "etcd_port-0-device-id",
			Peers:      []*voltha.Port_PeerPort{},
		},
	}

	TestProxy_Stats = &openflow_13.OfpFlowStats{
		Id: 1111,
	}
	TestProxy_Flows = &openflow_13.Flows{
		Items: []*openflow_13.OfpFlowStats{TestProxy_Stats},
	}
	TestProxy_Device = &voltha.Device{
		Id:         TestProxy_DeviceId,
		Type:       "simulated_olt",
		Address:    &voltha.Device_HostAndPort{HostAndPort: "1.2.3.4:5555"},
		AdminState: voltha.AdminState_PREPROVISIONED,
		Flows:      TestProxy_Flows,
		Ports:      TestProxy_Ports,
	}

	TestProxy_LogicalDevice = &voltha.LogicalDevice{
		Id:         TestProxy_DeviceId,
		DatapathId: 0,
		Ports:      TestProxy_LogicalPorts,
		Flows:      TestProxy_Flows,
	}
}

func TestProxy_1_1_1_Add_NewDevice(t *testing.T) {
	devIDBin, _ := uuid.New().MarshalBinary()
	TestProxy_DeviceId = "0001" + hex.EncodeToString(devIDBin)[:12]
	TestProxy_Device.Id = TestProxy_DeviceId

	preAddExecuted := false
	postAddExecuted := false

	devicesProxy := TestProxy_Root.node.CreateProxy("/devices", false)
	devicesProxy.RegisterCallback(PRE_ADD, commonCallback2, "PRE_ADD Device container changes")
	devicesProxy.RegisterCallback(POST_ADD, commonCallback2, "POST_ADD Device container changes")

	// Register ADD instructions callbacks
	TestProxy_Root_Device.RegisterCallback(PRE_ADD, commonCallback, "PRE_ADD instructions", &preAddExecuted)
	TestProxy_Root_Device.RegisterCallback(POST_ADD, commonCallback, "POST_ADD instructions", &postAddExecuted)

	// Add the device
	if added := TestProxy_Root_Device.Add("/devices", TestProxy_Device, ""); added == nil {
		t.Error("Failed to add device")
	} else {
		t.Logf("Added device : %+v", added)
	}

	// Verify that the added device can now be retrieved
	if d := TestProxy_Root_Device.Get("/devices/"+TestProxy_DeviceId, 0, false, ""); !reflect.ValueOf(d).IsValid() {
		t.Error("Failed to find added device")
	} else {
		djson, _ := json.Marshal(d)
		t.Logf("Found device: %s", string(djson))
	}

	if !preAddExecuted {
		t.Error("PRE_ADD callback was not executed")
	}
	if !postAddExecuted {
		t.Error("POST_ADD callback was not executed")
	}
}

func TestProxy_1_1_2_Add_ExistingDevice(t *testing.T) {
	TestProxy_Device.Id = TestProxy_DeviceId

	added := TestProxy_Root_Device.Add("/devices", TestProxy_Device, "");
	if added.(proto.Message).String() != reflect.ValueOf(TestProxy_Device).Interface().(proto.Message).String() {
		t.Errorf("Devices don't match - existing: %+v returned: %+v", TestProxy_LogicalDevice, added)
	}
}

func TestProxy_1_2_1_Get_AllDevices(t *testing.T) {
	devices := TestProxy_Root_Device.Get("/devices", 1, false, "")

	if len(devices.([]interface{})) == 0 {
		t.Error("there are no available devices to retrieve")
	} else {
		// Save the target device id for later tests
		TestProxy_TargetDeviceId = devices.([]interface{})[0].(*voltha.Device).Id
		t.Logf("retrieved all devices: %+v", devices)
	}
}

func TestProxy_1_2_2_Get_SingleDevice(t *testing.T) {
	if d := TestProxy_Root_Device.Get("/devices/"+TestProxy_TargetDeviceId, 0, false, ""); !reflect.ValueOf(d).IsValid() {
		t.Errorf("Failed to find device : %s", TestProxy_TargetDeviceId)
	} else {
		djson, _ := json.Marshal(d)
		t.Logf("Found device: %s", string(djson))
	}
}

func TestProxy_1_3_1_Update_Device(t *testing.T) {
	var fwVersion int
	preUpdateExecuted := false
	postUpdateExecuted := false

	if retrieved := TestProxy_Root_Device.Get("/devices/"+TestProxy_TargetDeviceId, 1, false, ""); retrieved == nil {
		t.Error("Failed to get device")
	} else {
		t.Logf("Found raw device (root proxy): %+v", retrieved)

		if retrieved.(*voltha.Device).FirmwareVersion == "n/a" {
			fwVersion = 0
		} else {
			fwVersion, _ = strconv.Atoi(retrieved.(*voltha.Device).FirmwareVersion)
			fwVersion++
		}

		retrieved.(*voltha.Device).FirmwareVersion = strconv.Itoa(fwVersion)

		TestProxy_Root_Device.RegisterCallback(
			PRE_UPDATE,
			commonCallback,
			"PRE_UPDATE instructions (root proxy)", &preUpdateExecuted,
		)
		TestProxy_Root_Device.RegisterCallback(
			POST_UPDATE,
			commonCallback,
			"POST_UPDATE instructions (root proxy)", &postUpdateExecuted,
		)

		if afterUpdate := TestProxy_Root_Device.Update("/devices/"+TestProxy_TargetDeviceId, retrieved, false, ""); afterUpdate == nil {
			t.Error("Failed to update device")
		} else {
			t.Logf("Updated device : %+v", afterUpdate)
		}

		if d := TestProxy_Root_Device.Get("/devices/"+TestProxy_TargetDeviceId, 1, false, ""); !reflect.ValueOf(d).IsValid() {
			t.Error("Failed to find updated device (root proxy)")
		} else {
			djson, _ := json.Marshal(d)
			t.Logf("Found device (root proxy): %s raw: %+v", string(djson), d)
		}

		if !preUpdateExecuted {
			t.Error("PRE_UPDATE callback was not executed")
		}
		if !postUpdateExecuted {
			t.Error("POST_UPDATE callback was not executed")
		}
	}
}

func TestProxy_1_3_2_Update_DeviceFlows(t *testing.T) {
	// Get a device proxy and update a specific port
	devFlowsProxy := TestProxy_Root.node.CreateProxy("/devices/"+TestProxy_DeviceId+"/flows", false)
	flows := devFlowsProxy.Get("/", 0, false, "")
	flows.(*openflow_13.Flows).Items[0].TableId = 2244

	preUpdateExecuted := false
	postUpdateExecuted := false

	devFlowsProxy.RegisterCallback(
		PRE_UPDATE,
		commonCallback,
		"PRE_UPDATE instructions (flows proxy)", &preUpdateExecuted,
	)
	devFlowsProxy.RegisterCallback(
		POST_UPDATE,
		commonCallback,
		"POST_UPDATE instructions (flows proxy)", &postUpdateExecuted,
	)

	kvFlows := devFlowsProxy.Get("/", 0, false, "")

	if reflect.DeepEqual(flows, kvFlows) {
		t.Errorf("Local changes have changed the KV store contents -  local:%+v, kv: %+v", flows, kvFlows)
	}

	if updated := devFlowsProxy.Update("/", flows.(*openflow_13.Flows), false, ""); updated == nil {
		t.Error("Failed to update flow")
	} else {
		t.Logf("Updated flows : %+v", updated)
	}

	if d := devFlowsProxy.Get("/", 0, false, ""); d == nil {
		t.Error("Failed to find updated flows (flows proxy)")
	} else {
		djson, _ := json.Marshal(d)
		t.Logf("Found flows (flows proxy): %s", string(djson))
	}

	if d := TestProxy_Root_Device.Get("/devices/"+TestProxy_DeviceId+"/flows", 1, false, ""); !reflect.ValueOf(d).IsValid() {
		t.Error("Failed to find updated flows (root proxy)")
	} else {
		djson, _ := json.Marshal(d)
		t.Logf("Found flows (root proxy): %s", string(djson))
	}

	if !preUpdateExecuted {
		t.Error("PRE_UPDATE callback was not executed")
	}
	if !postUpdateExecuted {
		t.Error("POST_UPDATE callback was not executed")
	}
}

func TestProxy_1_4_1_Remove_Device(t *testing.T) {
	preRemoveExecuted := false
	postRemoveExecuted := false

	TestProxy_Root_Device.RegisterCallback(
		PRE_REMOVE,
		commonCallback,
		"PRE_REMOVE instructions (root proxy)", &preRemoveExecuted,
	)
	TestProxy_Root_Device.RegisterCallback(
		POST_REMOVE,
		commonCallback,
		"POST_REMOVE instructions (root proxy)", &postRemoveExecuted,
	)

	if removed := TestProxy_Root_Device.Remove("/devices/"+TestProxy_DeviceId, ""); removed == nil {
		t.Error("Failed to remove device")
	} else {
		t.Logf("Removed device : %+v", removed)
	}
	if d := TestProxy_Root_Device.Get("/devices/"+TestProxy_DeviceId, 0, false, ""); reflect.ValueOf(d).IsValid() {
		djson, _ := json.Marshal(d)
		t.Errorf("Device was not removed - %s", djson)
	} else {
		t.Logf("Device was removed: %s", TestProxy_DeviceId)
	}

	if !preRemoveExecuted {
		t.Error("PRE_REMOVE callback was not executed")
	}
	if !postRemoveExecuted {
		t.Error("POST_REMOVE callback was not executed")
	}
}

func TestProxy_2_1_1_Add_NewLogicalDevice(t *testing.T) {

	ldIDBin, _ := uuid.New().MarshalBinary()
	TestProxy_LogicalDeviceId = "0001" + hex.EncodeToString(ldIDBin)[:12]
	TestProxy_LogicalDevice.Id = TestProxy_LogicalDeviceId

	preAddExecuted := false
	postAddExecuted := false

	// Register
	TestProxy_Root_LogicalDevice.RegisterCallback(PRE_ADD, commonCallback, "PRE_ADD instructions", &preAddExecuted)
	TestProxy_Root_LogicalDevice.RegisterCallback(POST_ADD, commonCallback, "POST_ADD instructions", &postAddExecuted)

	if added := TestProxy_Root_LogicalDevice.Add("/logical_devices", TestProxy_LogicalDevice, ""); added == nil {
		t.Error("Failed to add logical device")
	} else {
		t.Logf("Added logical device : %+v", added)
	}

	if ld := TestProxy_Root_LogicalDevice.Get("/logical_devices/"+TestProxy_LogicalDeviceId, 0, false, ""); !reflect.ValueOf(ld).IsValid() {
		t.Error("Failed to find added logical device")
	} else {
		ldJSON, _ := json.Marshal(ld)
		t.Logf("Found logical device: %s", string(ldJSON))
	}

	if !preAddExecuted {
		t.Error("PRE_ADD callback was not executed")
	}
	if !postAddExecuted {
		t.Error("POST_ADD callback was not executed")
	}
}

func TestProxy_2_1_2_Add_ExistingLogicalDevice(t *testing.T) {
	TestProxy_LogicalDevice.Id = TestProxy_LogicalDeviceId

	added := TestProxy_Root_LogicalDevice.Add("/logical_devices", TestProxy_LogicalDevice, "");
	if added.(proto.Message).String() != reflect.ValueOf(TestProxy_LogicalDevice).Interface().(proto.Message).String() {
		t.Errorf("Logical devices don't match - existing: %+v returned: %+v", TestProxy_LogicalDevice, added)
	}
}

func TestProxy_2_2_1_Get_AllLogicalDevices(t *testing.T) {
	logicalDevices := TestProxy_Root_LogicalDevice.Get("/logical_devices", 1, false, "")

	if len(logicalDevices.([]interface{})) == 0 {
		t.Error("there are no available logical devices to retrieve")
	} else {
		// Save the target device id for later tests
		TestProxy_TargetLogicalDeviceId = logicalDevices.([]interface{})[0].(*voltha.LogicalDevice).Id
		t.Logf("retrieved all logical devices: %+v", logicalDevices)
	}
}

func TestProxy_2_2_2_Get_SingleLogicalDevice(t *testing.T) {
	if ld := TestProxy_Root_LogicalDevice.Get("/logical_devices/"+TestProxy_TargetLogicalDeviceId, 0, false, ""); !reflect.ValueOf(ld).IsValid() {
		t.Errorf("Failed to find logical device : %s", TestProxy_TargetLogicalDeviceId)
	} else {
		ldJSON, _ := json.Marshal(ld)
		t.Logf("Found logical device: %s", string(ldJSON))
	}

}

func TestProxy_2_3_1_Update_LogicalDevice(t *testing.T) {
	var fwVersion int
	preUpdateExecuted := false
	postUpdateExecuted := false

	if retrieved := TestProxy_Root_LogicalDevice.Get("/logical_devices/"+TestProxy_TargetLogicalDeviceId, 1, false, ""); retrieved == nil {
		t.Error("Failed to get logical device")
	} else {
		t.Logf("Found raw logical device (root proxy): %+v", retrieved)

		if retrieved.(*voltha.LogicalDevice).RootDeviceId == "" {
			fwVersion = 0
		} else {
			fwVersion, _ = strconv.Atoi(retrieved.(*voltha.LogicalDevice).RootDeviceId)
			fwVersion++
		}

		TestProxy_Root_LogicalDevice.RegisterCallback(
			PRE_UPDATE,
			commonCallback,
			"PRE_UPDATE instructions (root proxy)", &preUpdateExecuted,
		)
		TestProxy_Root_LogicalDevice.RegisterCallback(
			POST_UPDATE,
			commonCallback,
			"POST_UPDATE instructions (root proxy)", &postUpdateExecuted,
		)

		retrieved.(*voltha.LogicalDevice).RootDeviceId = strconv.Itoa(fwVersion)

		if afterUpdate := TestProxy_Root_LogicalDevice.Update("/logical_devices/"+TestProxy_TargetLogicalDeviceId, retrieved, false,
			""); afterUpdate == nil {
			t.Error("Failed to update logical device")
		} else {
			t.Logf("Updated logical device : %+v", afterUpdate)
		}
		if d := TestProxy_Root_LogicalDevice.Get("/logical_devices/"+TestProxy_TargetLogicalDeviceId, 1, false, ""); !reflect.ValueOf(d).IsValid() {
			t.Error("Failed to find updated logical device (root proxy)")
		} else {
			djson, _ := json.Marshal(d)

			t.Logf("Found logical device (root proxy): %s raw: %+v", string(djson), d)
		}

		if !preUpdateExecuted {
			t.Error("PRE_UPDATE callback was not executed")
		}
		if !postUpdateExecuted {
			t.Error("POST_UPDATE callback was not executed")
		}
	}
}

func TestProxy_2_3_2_Update_LogicalDeviceFlows(t *testing.T) {
	// Get a device proxy and update a specific port
	ldFlowsProxy := TestProxy_Root.node.CreateProxy("/logical_devices/"+TestProxy_LogicalDeviceId+"/flows", false)
	flows := ldFlowsProxy.Get("/", 0, false, "")
	flows.(*openflow_13.Flows).Items[0].TableId = rand.Uint32()
	t.Logf("before updated flows: %+v", flows)

	ldFlowsProxy.RegisterCallback(
		PRE_UPDATE,
		commonCallback2,
	)
	ldFlowsProxy.RegisterCallback(
		POST_UPDATE,
		commonCallback2,
	)

	kvFlows := ldFlowsProxy.Get("/", 0, false, "")

	if reflect.DeepEqual(flows, kvFlows) {
		t.Errorf("Local changes have changed the KV store contents -  local:%+v, kv: %+v", flows, kvFlows)
	}

	if updated := ldFlowsProxy.Update("/", flows.(*openflow_13.Flows), false, ""); updated == nil {
		t.Error("Failed to update logical device flows")
	} else {
		t.Logf("Updated logical device flows : %+v", updated)
	}

	if d := ldFlowsProxy.Get("/", 0, false, ""); d == nil {
		t.Error("Failed to find updated logical device flows (flows proxy)")
	} else {
		djson, _ := json.Marshal(d)
		t.Logf("Found flows (flows proxy): %s", string(djson))
	}

	if d := TestProxy_Root_LogicalDevice.Get("/logical_devices/"+TestProxy_LogicalDeviceId+"/flows", 0, false,
		""); !reflect.ValueOf(d).IsValid() {
		t.Error("Failed to find updated logical device flows (root proxy)")
	} else {
		djson, _ := json.Marshal(d)
		t.Logf("Found logical device flows (root proxy): %s", string(djson))
	}
}

func TestProxy_2_4_1_Remove_Device(t *testing.T) {
	preRemoveExecuted := false
	postRemoveExecuted := false

	TestProxy_Root_LogicalDevice.RegisterCallback(
		PRE_REMOVE,
		commonCallback,
		"PRE_REMOVE instructions (root proxy)", &preRemoveExecuted,
	)
	TestProxy_Root_LogicalDevice.RegisterCallback(
		POST_REMOVE,
		commonCallback,
		"POST_REMOVE instructions (root proxy)", &postRemoveExecuted,
	)

	if removed := TestProxy_Root_LogicalDevice.Remove("/logical_devices/"+TestProxy_LogicalDeviceId, ""); removed == nil {
		t.Error("Failed to remove logical device")
	} else {
		t.Logf("Removed device : %+v", removed)
	}
	if d := TestProxy_Root_LogicalDevice.Get("/logical_devices/"+TestProxy_LogicalDeviceId, 0, false, ""); reflect.ValueOf(d).IsValid() {
		djson, _ := json.Marshal(d)
		t.Errorf("Device was not removed - %s", djson)
	} else {
		t.Logf("Device was removed: %s", TestProxy_LogicalDeviceId)
	}

	if !preRemoveExecuted {
		t.Error("PRE_REMOVE callback was not executed")
	}
	if !postRemoveExecuted {
		t.Error("POST_REMOVE callback was not executed")
	}
}

// -----------------------------
// Callback tests
// -----------------------------

func TestProxy_Callbacks_1_Register(t *testing.T) {
	TestProxy_Root_Device.RegisterCallback(PRE_ADD, firstCallback, "abcde", "12345")

	m := make(map[string]string)
	m["name"] = "fghij"
	TestProxy_Root_Device.RegisterCallback(PRE_ADD, secondCallback, m, 1.2345)

	d := &voltha.Device{Id: "12345"}
	TestProxy_Root_Device.RegisterCallback(PRE_ADD, thirdCallback, "klmno", d)
}

func TestProxy_Callbacks_2_Invoke_WithNoInterruption(t *testing.T) {
	TestProxy_Root_Device.InvokeCallbacks(PRE_ADD, false, nil)
}

func TestProxy_Callbacks_3_Invoke_WithInterruption(t *testing.T) {
	TestProxy_Root_Device.InvokeCallbacks(PRE_ADD, true, nil)
}

func TestProxy_Callbacks_4_Unregister(t *testing.T) {
	TestProxy_Root_Device.UnregisterCallback(PRE_ADD, firstCallback)
	TestProxy_Root_Device.UnregisterCallback(PRE_ADD, secondCallback)
	TestProxy_Root_Device.UnregisterCallback(PRE_ADD, thirdCallback)
}

//func TestProxy_Callbacks_5_Add(t *testing.T) {
//	TestProxy_Root_Device.Root.AddCallback(TestProxy_Root_Device.InvokeCallbacks, POST_UPDATE, false, "some data", "some new data")
//}
//
//func TestProxy_Callbacks_6_Execute(t *testing.T) {
//	TestProxy_Root_Device.Root.ExecuteCallbacks()
//}
