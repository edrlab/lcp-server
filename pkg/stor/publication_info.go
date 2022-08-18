// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"errors"

	"gorm.io/gorm"
)

// PublicationInfo data model
type PublicationInfo struct {
	gorm.Model
	UUID          string `json:"uuid" gorm:"uniqueIndex"`
	Title         string `json:"title,omitempty"`
	EncryptionKey []byte `json:"encryption_key"`
	Location      string `json:"location"`
	ContentType   string `json:"content_type"`
	Size          uint32 `json:"size"`
	Checksum      string `json:"checksum"`
}

// Validate checks required fields and values
func (p *PublicationInfo) Validate() error {

	if p.UUID == "" {
		return errors.New("required field missing: UUID")
	}
	if p.Title == "" {
		return errors.New("required field missing: Title")
	}
	return nil
}

func (s publicationStore) ListAll() (*[]PublicationInfo, error) {
	publications := []PublicationInfo{}
	// security: limited to 1000 results
	return &publications, s.db.Limit(1000).Order("id ASC").Find(&publications).Error
}

func (s publicationStore) List(pageSize, pageNum int) (*[]PublicationInfo, error) {
	publications := []PublicationInfo{}
	// pageNum starts at 1
	// result sorted to assure the same order for each request
	return &publications, s.db.Offset((pageNum - 1) * pageSize).Limit(pageSize).Order("id ASC").Find(&publications).Error
}

func (s publicationStore) FindByType(contentType string) (*[]PublicationInfo, error) {
	publications := []PublicationInfo{}
	return &publications, s.db.Limit(1000).Find(&publications, "content_type= ?", contentType).Error
}

func (s publicationStore) Count() (int64, error) {
	var count int64
	return count, s.db.Model(PublicationInfo{}).Count(&count).Error
}

func (s publicationStore) Get(uuid string) (*PublicationInfo, error) {
	var publication PublicationInfo
	return &publication, s.db.Where("uuid = ?", uuid).First(&publication).Error
}

func (s publicationStore) Create(newPublication *PublicationInfo) error {
	return s.db.Create(newPublication).Error
}

func (s publicationStore) Update(changedPublication *PublicationInfo) error {
	return s.db.Save(changedPublication).Error
}

func (s publicationStore) Delete(deletedPublication *PublicationInfo) error {
	return s.db.Delete(deletedPublication).Error
}
