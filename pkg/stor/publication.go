// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"time"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// Publication data model
type Publication struct {
	gorm.Model
	CreatedAt     time.Time `gorm:"index"` // index on created_at, useful for dashboard queries
	Provider      string    `json:"provider,omitempty" validate:"omitempty,url" gorm:"type:varchar(255)"`
	UUID          string    `json:"uuid" validate:"omitempty,uuid" gorm:"type:varchar(100);uniqueIndex"`
	AltID         string    `json:"alt_id,omitempty" validate:"omitempty" gorm:"type:varchar(255);index"`
	ContentType   string    `json:"content_type" validate:"required" gorm:"type:varchar(100);index"`
	Title         string    `json:"title" validate:"required"`
	Description   string    `json:"description,omitempty"`
	Authors       string    `json:"authors,omitempty"`
	Publishers    string    `json:"publishers,omitempty"`
	CoverUrl      string    `json:"cover_url,omitempty" validate:"omitempty,url" gorm:"type:varchar(1024)"`
	EncryptionKey []byte    `json:"encryption_key" validate:"required"`
	Href          string    `json:"href" validate:"required,http_url" gorm:"type:varchar(1024)"`
	Size          uint32    `json:"size" validate:"required,number"`
	Checksum      string    `json:"checksum" validate:"required,base64" gorm:"type:varchar(255)"`
}

// Validate checks required fields and values
func (p *Publication) Validate() error {

	validate := validator.New()
	return validate.Struct(p)
}

func (s publicationStore) ListAll() (*[]Publication, error) {
	publications := []Publication{}
	// security: limited to 1000 results, in descending order of ID to have a stable order
	return &publications, s.db.Limit(1000).Order("id DESC").Find(&publications).Error
}

func (s publicationStore) List(pageNum, pageSize int) (*[]Publication, error) {
	publications := []Publication{}
	// pageNum starts at 1
	// result sorted to assure the same order for each request
	return &publications, s.db.Offset((pageNum - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&publications).Error
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

func (s publicationStore) GetByAltID(altID string) (*Publication, error) {
	var publication Publication
	return &publication, s.db.Where("alt_id = ?", altID).First(&publication).Error
}
