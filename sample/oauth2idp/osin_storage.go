package main

import (
	"github.com/RangelReale/osin"
)

var _ osin.Storage = &OsinOnMemoryStorage{}

type OsinOnMemoryStorage struct {
	clients   map[string]osin.Client
	authorize map[string]*osin.AuthorizeData
	access    map[string]*osin.AccessData
	refresh   map[string]string
}

func (s *OsinOnMemoryStorage) Clone() osin.Storage {
	return s
}

func (s *OsinOnMemoryStorage) Close() {
}

func (s *OsinOnMemoryStorage) GetClient(id string) (osin.Client, error) {
	if c, ok := s.clients[id]; ok {
		return c, nil
	}
	return nil, osin.ErrNotFound
}

func (s *OsinOnMemoryStorage) SetClient(id string, client osin.Client) error {
	s.clients[id] = client
	return nil
}

func (s *OsinOnMemoryStorage) SaveAuthorize(data *osin.AuthorizeData) error {
	s.authorize[data.Code] = data
	return nil
}

func (s *OsinOnMemoryStorage) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	if d, ok := s.authorize[code]; ok {
		return d, nil
	}
	return nil, osin.ErrNotFound
}

func (s *OsinOnMemoryStorage) RemoveAuthorize(code string) error {
	delete(s.authorize, code)
	return nil
}

func (s *OsinOnMemoryStorage) SaveAccess(data *osin.AccessData) error {
	s.access[data.AccessToken] = data
	if data.RefreshToken != "" {
		s.refresh[data.RefreshToken] = data.AccessToken
	}
	return nil
}

func (s *OsinOnMemoryStorage) LoadAccess(code string) (*osin.AccessData, error) {
	if d, ok := s.access[code]; ok {
		return d, nil
	}
	return nil, osin.ErrNotFound
}

func (s *OsinOnMemoryStorage) RemoveAccess(code string) error {
	delete(s.access, code)
	return nil
}

func (s *OsinOnMemoryStorage) LoadRefresh(code string) (*osin.AccessData, error) {
	if d, ok := s.refresh[code]; ok {
		return s.LoadAccess(d)
	}
	return nil, osin.ErrNotFound
}

func (s *OsinOnMemoryStorage) RemoveRefresh(code string) error {
	delete(s.refresh, code)
	return nil
}
