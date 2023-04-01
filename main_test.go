package main

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	ctl *gomock.Controller

	scanner *MockTextScanner
}

func (s *Suite) SetupTest() {
	s.ctl = gomock.NewController(s.T())

	s.scanner = NewMockTextScanner(s.ctl)
}

func (s *Suite) TeardownTest() {
	s.ctl.Finish()
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func Test_emptyTextScanner(t *testing.T) {
	count, freq := countSearchQueriesFreq(nil)

	assert.Equal(t, 0, count)
	assert.Len(t, freq, 0)
}

func (s *Suite) Test_countSearchQueriesFreq() {
	tt := []struct {
		Name      string
		ApplyMock func()
		ExpRows   int
		ExpFreq   map[string]*freq
	}{
		{
			"read from text scanner without memory limit",
			func() {
				s.scanner.EXPECT().Scan().Return(true).Times(3)
				s.scanner.EXPECT().Scan().Return(false).Times(1)

				s.scanner.EXPECT().Text().Return("new").Times(1)
				s.scanner.EXPECT().Text().Return("test").Times(2)
			},
			3,
			map[string]*freq{
				"test": {
					2, 2,
				},
				"new": {
					1, 1,
				},
			},
		},
		{
			"read from text scanner with memory limit",
			func() {
				s.scanner.EXPECT().Scan().Return(true).Times(1)

				s.scanner.EXPECT().Text().Return("new").Times(1)
			},
			1,
			map[string]*freq{
				"new": {
					1, 1,
				},
			},
		},
	}

	for _, tc := range tt {
		tc := tc

		s.T().Run(tc.Name, func(t *testing.T) {
			tc.ApplyMock()

			count, freq := countSearchQueriesFreq(s.scanner)

			assert.Equal(t, tc.ExpRows, count)
			assert.Equal(t, tc.ExpFreq, freq)
		})
	}
}
