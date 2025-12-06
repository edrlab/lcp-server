// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package api

import (
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListPublications lists publications present in the database.
func (a *APICtrl) ListPublications(w http.ResponseWriter, r *http.Request) {
	log.Debug("List Publications")

	page := r.Context().Value(PageKey).(int)
	perPage := r.Context().Value(PerPageKey).(int)

	var publications *[]stor.Publication
	var err error

	if page == 0 || perPage == 0 {
		publications, err = a.Store.Publication().ListAll()
	} else {
		publications, err = a.Store.Publication().List(page, perPage)
	}
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	if err := render.RenderList(w, r, NewPublicationListResponse(publications)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// SearchPublications searches publications corresponding to a specific criteria.
func (a *APICtrl) SearchPublications(w http.ResponseWriter, r *http.Request) {
	log.Debug("Search Publications ")

	var publications *[]stor.Publication
	var err error

	// by format
	if format := r.URL.Query().Get("format"); format != "" {
		var contentType string
		switch format {
		case "epub":
			contentType = "application/epub+zip"
		case "pdf":
			contentType = "application/pdf"
		case "lcpdf":
			contentType = "application/pdf+lcp"
		case "lcpau":
			contentType = "application/audiobook+lcp"
		case "lcpdi":
			contentType = "application/divina+lcp"
		default:
			err = errors.New("invalid content type query string parameter")
		}
		if contentType != "" {
			publications, err = a.Store.Publication().FindByType(contentType)
		}
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("invalid format parameter")))
		return
	}
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	if err := render.RenderList(w, r, NewPublicationListResponse(publications)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// CreatePublication adds a new Publication to the database.
func (a *APICtrl) CreatePublication(w http.ResponseWriter, r *http.Request) {

	// get the payload
	data := &PublicationRequest{}
	if err := render.Bind(r, data); err != nil {
		log.Errorf("Create Publication: unable to bind the json request: %v", err)
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	publication := data.Publication

	// Check the presence of a UUID
	if publication.UUID == "" {
		log.Error("Create Publication: missing required publication UUID")
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required publication ID")))
		return
	}

	// db create
	err := a.Store.Publication().Create(publication)
	if err != nil {
		log.Errorf("Create Publication: failed to create publication: %v", err)
		render.Render(w, r, ErrServer(err))
		return
	}

	log.Debug("Create Publication ", publication.Title)

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, NewPublicationResponse(publication)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// GetPublication returns a specific publication
func (a *APICtrl) GetPublication(w http.ResponseWriter, r *http.Request) {
	log.Debug("Get Publication")

	var publication *stor.Publication
	var err error

	if publicationID := chi.URLParam(r, "publicationID"); publicationID != "" {
		log.Debugf("Get Publication: %s", publicationID)
		publication, err = a.Store.Publication().Get(publicationID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required publication ID")))
	}
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(errors.New("invalid publication ID")))
		return
	}
	if err := render.Render(w, r, NewPublicationResponse(publication)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// UpdatePublication updates an existing Publication in the database.
func (a *APICtrl) UpdatePublication(w http.ResponseWriter, r *http.Request) {
	log.Debug("Update Publication")

	// get the payload
	data := &PublicationRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	pubUpdates := data.Publication

	var publication *stor.Publication
	var err error

	// get the existing publication
	if publicationID := chi.URLParam(r, "publicationID"); publicationID != "" {
		publication, err = a.Store.Publication().Get(publicationID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required publication ID"))) // publicationID is nil
		return
	}
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(errors.New("invalid publication ID")))
		return
	}

	// set updated fields
	publication.Provider = pubUpdates.Provider
	publication.Title = pubUpdates.Title
	publication.Authors = pubUpdates.Authors
	publication.CoverUrl = pubUpdates.CoverUrl
	publication.EncryptionKey = pubUpdates.EncryptionKey
	publication.Href = pubUpdates.Href
	publication.ContentType = pubUpdates.ContentType
	publication.Size = pubUpdates.Size
	publication.Checksum = pubUpdates.Checksum

	// db update
	err = a.Store.Publication().Update(publication)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}

	if err := render.Render(w, r, NewPublicationResponse(publication)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// DeletePublication removes an existing Publication from the database.
func (a *APICtrl) DeletePublication(w http.ResponseWriter, r *http.Request) {
	log.Debug("Delete Publication")

	var publication *stor.Publication
	var err error

	// get the existing publication
	if publicationID := chi.URLParam(r, "publicationID"); publicationID != "" {
		log.Debugf("Delete Publication: %s", publicationID)
		publication, err = a.Store.Publication().Get(publicationID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required publication ID"))) // publicationID is nil
		return
	}
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(errors.New("invalid publication ID")))
		return
	}

	// db delete
	err = a.Store.Publication().Delete(publication)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}

	if err := render.Render(w, r, NewPublicationResponse(publication)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// --
// Request and Response payloads for the REST api.
// --

type omit *struct{}

// PublicationRequest is the request publication payload.
type PublicationRequest struct {
	*stor.Publication
}

// PublicationResponse is the response publication payload.
type PublicationResponse struct {
	*stor.Publication
	// TODO do not serialize the following properties
	//ID omit `json:"ID,omitempty"`
	//CreatedAt omit `json:"CreatedAt,omitempty"`
	//UpdatedAt omit `json:"UpdatedAt,omitempty"`
	//DeletedAt omit `json:"DeletedAt,omitempty"`
}

// NewPublicationListResponse creates a rendered list of publications
func NewPublicationListResponse(publications *[]stor.Publication) []render.Renderer {
	list := []render.Renderer{}
	for i := 0; i < len(*publications); i++ {
		list = append(list, NewPublicationResponse(&(*publications)[i]))
	}
	return list
}

// NewPublicationResponse creates a rendered publication.
func NewPublicationResponse(pub *stor.Publication) *PublicationResponse {
	return &PublicationResponse{Publication: pub}
}

// Bind post-processes requests after unmarshalling.
func (p *PublicationRequest) Bind(r *http.Request) error {
	return p.Publication.Validate()
}

// Render processes responses before marshalling.
func (pub *PublicationResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
