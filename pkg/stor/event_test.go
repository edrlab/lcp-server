package stor

import (
	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	var err error

	// select a publication and a license
	l := Licenses[0]
	pid := l.PublicationID
	var p Publication
	found := false
	for _, p = range Publications {
		if p.UUID == pid {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Failed to get the publication associated with a license: %v", err)
	}

	// create an event
	now := time.Now().Truncate(time.Second)
	e1 := &Event{
		Timestamp:  now,
		Type:       "register",
		DeviceName: "Test Device Name 1",
		DeviceID:   "Test Device ID 1",
		LicenseID:  l.UUID,
	}

	err = St.Event().Create(e1)
	if err != nil {
		t.Fatalf("Failed to create an event: %v", err)
	}

	// get the event
	var event *Event
	event, err = St.Event().Get(e1.ID)
	if err != nil {
		t.Fatalf("Failed to create an event: %v", err)
	}
	if event.Type != e1.Type {
		t.Fatalf("Event modified when retrieved")
	}

	// create a second event
	now = time.Now().Truncate(time.Second)
	e2 := &Event{
		Timestamp:  now,
		Type:       "register",
		DeviceName: "Test Device Name 2",
		DeviceID:   "Test Device ID 2",
		LicenseID:  l.UUID,
	}

	err = St.Event().Create(e2)
	if err != nil {
		t.Fatalf("Failed to create an event: %v", err)
	}

	// count events
	var count int64
	count, err = St.Event().Count(l.UUID)
	if err != nil {
		t.Fatalf("Failed to count events: %v", err)
	}
	if count != 2 {
		t.Fatalf("Failed to count, expected 2 got %d", count)
	}

	// list events
	var events *[]Event
	events, err = St.Event().List(l.UUID)
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}
	if len(*events) != 2 {
		t.Fatalf("Failed to list, expected 2 got %d", count)
	}

	// get the register event associated with the first device
	event, err = St.Event().GetRegisterByDevice(l.UUID, e1.DeviceID)
	if err != nil {
		t.Fatalf("Failed to get event 1: %v", err)
	}
	if event.DeviceName != "Test Device Name 1" {
		t.Fatalf("Failed to retrieve the expected event, got : %s", event.DeviceName)
	}

	// try to get a non existant event
	deviceID := "Test Device ID 3"
	_, err = St.Event().GetRegisterByDevice(l.UUID, deviceID)
	if err == nil {
		t.Fatal("Failed to get an error for device id 3")
	}

	// update the first event
	e1.Type = "revoke"
	err = St.Event().Update(e1)
	if err != nil {
		t.Fatalf("Failed to update an event: %v", err)
	}

	// delete the events
	err = St.Event().Delete(e1)
	if err != nil {
		t.Fatalf("Failed to delete event 1: %v", err)
	}
	err = St.Event().Delete(e2)
	if err != nil {
		t.Fatalf("Failed to delete event 2: %v", err)
	}

	// count events again
	count, err = St.Event().Count(l.UUID)
	if err != nil {
		t.Fatalf("Failed to count events: %v", err)
	}
	if count != 0 {
		t.Fatalf("Failed to count, expected 0 got %d", count)
	}
}
