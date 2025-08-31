// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// Publication data model
type Publication struct {
	gorm.Model
	UUID          string `json:"uuid" validate:"omitempty,uuid" gorm:"uniqueIndex"`
	Title         string `json:"title,omitempty" validate:"required"`
	Authors       string `json:"authors,omitempty"`
	CoverUrl      string `json:"cover_url,omitempty" validate:"omitempty,url"`
	EncryptionKey []byte `json:"encryption_key" validate:"required"`
	Href          string `json:"href" validate:"required,http_url"`
	ContentType   string `json:"content_type" validate:"required"`
	Size          uint32 `json:"size" validate:"required,number"`
	Checksum      string `json:"checksum" validate:"required,base64"`
}

// Validate checks required fields and values
func (p *Publication) Validate() error {

	validate := validator.New()
	return validate.Struct(p)
}

func (s publicationStore) ListAll() (*[]Publication, error) {
	publications := []Publication{}
	// security: limited to 1000 results
	return &publications, s.db.Limit(1000).Order("id ASC").Find(&publications).Error
}

func (s publicationStore) List(pageNum, pageSize int) (*[]Publication, error) {
	publications := []Publication{}
	// pageNum starts at 1
	// result sorted to assure the same order for each request
	return &publications, s.db.Offset((pageNum - 1) * pageSize).Limit(pageSize).Order("id ASC").Find(&publications).Error
}

func (s publicationStore) FindByType(contentType string) (*[]Publication, error) {
	publications := []Publication{}
	return &publications, s.db.Limit(1000).Find(&publications, "content_type= ?", contentType).Error
}

func (s publicationStore) Count() (int64, error) {
	var count int64
	return count, s.db.Model(Publication{}).Count(&count).Error
}

func (s publicationStore) Get(uuid string) (*Publication, error) {
	var publication Publication
	return &publication, s.db.Where("uuid = ?", uuid).First(&publication).Error
}

func (s publicationStore) Create(newPublication *Publication) error {
	return s.db.Create(newPublication).Error
}

func (s publicationStore) Update(changedPublication *Publication) error {
	return s.db.Save(changedPublication).Error
}

func (s publicationStore) Delete(deletedPublication *Publication) error {
	return s.db.Delete(deletedPublication).Error
}
