package app

import "bitbucket.org/okteto/okteto/backend/model"

// PingDB will run a count query on the project table
// and return an error if unsuccessful
func (s *Server) PingDB() error {
	var c int
	result := s.DB.Model(&model.Project{}).Count(&c)
	return result.Error
}
