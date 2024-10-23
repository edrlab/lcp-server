// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"errors"
	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
	log "github.com/sirupsen/logrus"
)

var (
	ErrLicenseNotFound = errors.New("license not found or failed to get license info")
)

// StatusDoc data model
type (
	StatusDoc struct {
		ID              string           `json:"id"`
		Status          string           `json:"status"`
		Message         string           `json:"message"`
		Updated         Updated          `json:"updated"`
		Links           []Link           `json:"links"`
		PotentialRights *PotentialRights `json:"potential_rights,omitempty"`
		Events          []stor.Event     `json:"events,omitempty"`
	}

	Updated struct {
		License time.Time `json:"license"`
		Status  time.Time `json:"status"`
	}

	PotentialRights struct {
		End *time.Time `json:"end,omitempty"`
	}

	// License management interface
	LicenseManager interface {
		Register(license *stor.LicenseInfo) error
		Renew(license *stor.LicenseInfo) error
		Return(license *stor.LicenseInfo) error
		Revoke(license *stor.LicenseInfo) error
	}

	LicenseCtrl struct {
		*conf.Config // TODO: change for an interface (dependency)
		stor.Store
	}

	DeviceInfo struct {
		ID   string
		Name string
	}
)

func NewLicenseCtrl(cf *conf.Config, st stor.Store) *LicenseCtrl {
	return &LicenseCtrl{
		Config: cf,
		Store:  st,
	}
}

// ====

// NewStatusDoc returns a Status Document
func (lc *LicenseCtrl) NewStatusDoc(license *stor.LicenseInfo) *StatusDoc {

	// TODO: if the date of update of the publication is more recent than the date of creation of the license,
	// the content has been updated: the date of update of the license must therefore be set to the date of update
	// of the publication

	// set license updated
	var licUpdated, statUpdated time.Time
	if license.Updated != nil {
		licUpdated = *license.Updated
	} else {
		licUpdated = license.CreatedAt
	}
	if license.StatusUpdated != nil {
		statUpdated = *license.StatusUpdated
	} else {
		statUpdated = licUpdated
	}

	// set the status document
	statusDoc := &StatusDoc{
		ID:      license.UUID,
		Status:  license.Status,
		Message: "The license is in " + license.Status + " state", // TODO: flexible, localize
		Updated: Updated{
			License: licUpdated,
			Status:  statUpdated,
		},
	}

	// check if the license has expired
	now := time.Now().Truncate(time.Second)
	if (license.Status == stor.STATUS_READY || license.Status == stor.STATUS_ACTIVE) && now.After(*license.End) {
		statusDoc.Status = stor.STATUS_EXPIRED
		statusDoc.Message = "The license has expired on " + license.End.Format(time.RFC822)
	}

	// not need to return a max end date if the license is not ready or active
	if license.Status != stor.STATUS_READY && license.Status != stor.STATUS_ACTIVE {
		license.MaxEnd = nil
	}

	// set the max end date
	if license.MaxEnd != nil {
		potentialRights := &PotentialRights{
			End: license.MaxEnd,
		}
		statusDoc.PotentialRights = potentialRights
	}

	// set links
	setStatusLinks(lc.Config.PublicBaseUrl, lc.Config.Status.RenewLink, statusDoc)

	// set events
	setEvents(lc.Store, statusDoc)

	return statusDoc
}

// Set status links
func setStatusLinks(publicBaseUrl string, renewLink string, statusDoc *StatusDoc) error {
	var links []Link
	actions := [3]string{"register", "renew", "return"}

	for _, action := range actions {
		var href string
		//the provider can manage his own renew URL and take care of calling the license status server
		if action == "renew" {
			if renewLink != "" {
				href = renewLink + "{?end,id,name}"
			} else {
				href = publicBaseUrl + "/" + action + "/" + statusDoc.ID + "{?end,id,name}"
			}
		} else {
			href = publicBaseUrl + "/" + action + "/" + statusDoc.ID + "{?id,name}"
		}
		link := Link{Href: href, Rel: action, Type: ContentType_LSD_JSON, Templated: true}
		links = append(links, link)
	}

	// add the structure to the status document
	statusDoc.Links = links
	return nil
}

// Set events
func setEvents(store stor.Store, statusDoc *StatusDoc) error {

	events, err := store.Event().List(statusDoc.ID)
	if err != nil {
		return err
	}
	statusDoc.Events = *events
	return nil
}

// Register records that a new device is using a license
func (lc *LicenseCtrl) Register(licenseID string, device *DeviceInfo) (*StatusDoc, error) {

	// Get license info
	license, err := lc.Store.License().Get(licenseID)
	if err != nil {
		return nil, ErrLicenseNotFound
	}

	// check that the license is in ready or active status
	if (license.Status != stor.STATUS_ACTIVE) && (license.Status != stor.STATUS_READY) {
		return nil, errors.New("registering a device on an license that is neither ready nor active is not allowed")
	}

	// check that the device has not already been registered for this license
	_, err = lc.Store.Event().GetRegisterByDevice(license.UUID, device.ID)
	if err == nil {
		log.Warningf("Registration halted: the device %s is already registered", device.ID)
		statusDoc := lc.NewStatusDoc(license)
		return statusDoc, nil
	}

	// update the status document in the db
	if license.Status == stor.STATUS_READY {
		license.Status = stor.STATUS_ACTIVE
	}
	license.DeviceCount++
	now := time.Now().Truncate(time.Second)
	license.StatusUpdated = &now
	lc.Store.License().Update(license)

	// create an event
	event := &stor.Event{
		Timestamp:  now,
		Type:       stor.EVENT_REGISTER,
		DeviceID:   device.ID,
		DeviceName: device.Name,
		LicenseID:  licenseID,
	}

	err = lc.Store.Event().Create(event)
	if err != nil {
		log.Errorf("Failed to create an event: %v", err)
		return nil, err
	}

	statusDoc := lc.NewStatusDoc(license)
	return statusDoc, nil
}

// Renew extends the end date of a license
func (lc *LicenseCtrl) Renew(licenseID string, device *DeviceInfo, newEnd *time.Time) (*StatusDoc, error) {

	// Get license info
	license, err := lc.Store.License().Get(licenseID)
	if err != nil {
		return nil, ErrLicenseNotFound
	}

	// check that the license is in active status
	if license.Status != stor.STATUS_ACTIVE {
		return nil, errors.New("requesting a renew on a non-active license is prohibited")
	}

	// check that the device had been registered for this license
	_, err = lc.Store.Event().GetRegisterByDevice(license.UUID, device.ID)
	if err != nil {
		return nil, errors.New("requesting a renew on a license which has not been registered by this device is prohibited")
	}

	// set the new end date
	if newEnd != nil {
		// consider an explicit end date
		if license.MaxEnd != nil && newEnd.After(*license.MaxEnd) {
			log.Println("License extension limit is ", license.MaxEnd.Format(time.RFC822))
			license.End = license.MaxEnd
		} else {
			license.End = newEnd
		}
		// consider a default end date set in the configuration file
	} else if lc.Config.Status.RenewDefaultDays != 0 {
		*license.End = license.End.AddDate(0, 0, lc.Config.Status.RenewDefaultDays)
		// the ultimate default is 7 days
	} else {
		*license.End = license.End.AddDate(0, 0, 7)
	}
	log.Println("License extension; the new end date is ", license.End.Format(time.RFC822))

	// update the license in the db
	now := time.Now().Truncate(time.Second)
	license.Updated = &now
	lc.Store.License().Update(license)

	// create an event
	event := &stor.Event{
		Timestamp:  now,
		Type:       stor.EVENT_RENEW,
		DeviceID:   device.ID,
		DeviceName: device.Name,
		LicenseID:  licenseID,
	}

	err = lc.Store.Event().Create(event)
	if err != nil {
		log.Errorf("Failed to create an event: %v", err)
		return nil, err
	}

	statusDoc := lc.NewStatusDoc(license)
	return statusDoc, nil
}

// Return forces the expiration of a license and returns a status document.
func (lc *LicenseCtrl) Return(licenseID string, device *DeviceInfo) (*StatusDoc, error) {

	// Get license info
	license, err := lc.Store.License().Get(licenseID)
	if err != nil {
		return nil, ErrLicenseNotFound
	}

	// check that the license has not already expired
	now := time.Now().Truncate(time.Second)
	if license.End.Before(now) {
		return nil, errors.New("this license expired on " + license.End.Format(time.RFC822))
	}

	// check that the license is in active status
	if license.Status != stor.STATUS_ACTIVE {
		return nil, errors.New("requesting a return on a non-active license is prohibited")
	}

	// check that the device had been registered for this license
	_, err = lc.Store.Event().GetRegisterByDevice(license.UUID, device.ID)
	if err != nil {
		return nil, errors.New("requesting a return on a license which has not been registered by this device is prohibited")
	}

	// set the new end date
	license.End = &now

	log.Println("License returned; the new end date is ", license.End.Format(time.RFC822))

	// update the license and status document in the db
	license.Updated = &now
	license.Status = stor.STATUS_RETURNED
	license.StatusUpdated = &now
	lc.Store.License().Update(license)

	// create an event
	event := &stor.Event{
		Timestamp:  now,
		Type:       stor.EVENT_RETURN,
		DeviceID:   device.ID,
		DeviceName: device.Name,
		LicenseID:  licenseID,
	}

	err = lc.Store.Event().Create(event)
	if err != nil {
		log.Errorf("Failed to create an event: %v", err)
		return nil, err
	}

	statusDoc := lc.NewStatusDoc(license)
	return statusDoc, nil
}

// Revoke forces the expiration of a license and returns a status document.
func (lc *LicenseCtrl) Revoke(licenseID string) (*StatusDoc, error) {

	// Get license info
	license, err := lc.Store.License().Get(licenseID)
	if err != nil {
		return nil, ErrLicenseNotFound
	}

	if license.Status == stor.STATUS_REVOKED || license.Status == stor.STATUS_CANCELLED {
		log.Infof("The status of the license is already %s", license.Status)
		statusDoc := lc.NewStatusDoc(license)
		return statusDoc, nil
	}

	// check if the license is in ready status (-> cancel or revoke)
	cancel := false
	if license.Status == stor.STATUS_READY {
		cancel = true
	}

	// set the new end date
	now := time.Now().Truncate(time.Second)
	license.End = &now

	log.Println("License revoked or cancelled; the new end date is ", license.End.Format(time.RFC822))

	// update the license and status document in the db
	license.Updated = &now
	if cancel {
		license.Status = stor.STATUS_CANCELLED
	} else {
		license.Status = stor.STATUS_REVOKED
	}
	license.StatusUpdated = &now
	lc.Store.License().Update(license)

	// create an event
	event := &stor.Event{
		Timestamp:  now,
		Type:       stor.EVENT_REVOKE,
		DeviceID:   "admin",
		DeviceName: "system",
		LicenseID:  licenseID,
	}
	if cancel {
		event.Type = stor.EVENT_CANCEL
	} else {
		event.Type = stor.EVENT_REVOKE
	}

	err = lc.Store.Event().Create(event)
	if err != nil {
		log.Errorf("Failed to create an event: %v", err)
		return nil, err
	}

	statusDoc := lc.NewStatusDoc(license)
	return statusDoc, nil
}
