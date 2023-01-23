package stor

import (
	"testing"
	"time"
)

func TestEvent(t *testing.T) {
	var err error

	// store a publication and a license
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
	err = St.Publication().Create(&p)
	if err != nil {
		t.Fatalf("Failed to store a publication: %v", err)
	}
	err = St.License().Create(&l)
	if err != nil {
		t.Fatalf("Failed to store a license: %v", err)
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

	// update the first event
	e1.Type = "revoke"
	err = St.Event().Update(e1)
	if err != nil {
		t.Fatalf("Failed to update an event: %v", err)
	}

	// get one of the events
	event, err = St.Event().GetByDevice(l.UUID, e1.DeviceID)
	if err != nil {
		t.Fatalf("Failed to get event 1: %v", err)
	}
	if event.DeviceName != "Test Device Name 1" {
		t.Fatalf("Failed to retrieve the expected event, got : %s", event.DeviceName)
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

	// delete the license and publication
	err = St.License().Delete(&l)
	if err != nil {
		t.Fatalf("Failed to delete the license: %v", err)
	}
	err = St.Publication().Delete(&p)
	if err != nil {
		t.Fatalf("Failed to delete the publication: %v", err)
	}

}
