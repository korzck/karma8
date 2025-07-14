//go:build integration

package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestServiceSuite(t *testing.T) {
	suite.Run(t, &Suite{})
}

func (s *Suite) TestService() {
	s.Run("test basic", func() {
		db := initDB()
		_ = newTestService(db)

		var res string
		err := db.GetContext(context.Background(), &res, "SELECT 'karma8'")
		s.Require().NoError(err)

		s.Equal("karma8", res)
		fmt.Println("karma8")
		// дальше в том же духе, можно применять миграции, делать тесты, проверять логику
	})
}
