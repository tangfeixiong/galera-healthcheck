package sequence_number_test

import (
	"database/sql"
	"database/sql/driver"
	"errors"

	"github.com/erikstmartin/go-testdb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/mysqld_cmd/fakes"
	"github.com/cloudfoundry-incubator/galera-healthcheck/sequence_number"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("GaleraSequenceChecker", func() {

	const expectedSeqNumber = "32"

	var (
		sequenceChecker sequence_number.SequenceNumberChecker
		mysqldCmd       *fakes.FakeMysqldCmd
		rootConfig      config.Config
		logger          *lagertest.TestLogger
		db              *sql.DB
	)

	BeforeEach(func() {
		rootConfig = config.Config{}
		logger = lagertest.NewTestLogger("sequence_number test")
		db, _ = sql.Open("testdb", "")

		mysqldCmd = &fakes.FakeMysqldCmd{}
		mysqldCmd.RecoverSeqnoReturns(expectedSeqNumber, nil)
	})

	JustBeforeEach(func() {
		sequenceChecker = sequence_number.New(db, mysqldCmd, rootConfig, logger)
	})

	AfterEach(func() {
		testdb.Reset()
	})

	Describe("Check", func() {
		Context("db works", func() {

			BeforeEach(func() {
				testdb.SetExecFunc(func(query string) (driver.Result, error) {
					return nil, nil
				})
			})

			It("returns an unsuccessful check", func() {
				_, err := sequenceChecker.Check()
				Expect(err).To(MatchError("can't determine sequence number when database is running"))
			})
		})

		Context("db is down", func() {
			BeforeEach(func() {
				testdb.SetExecFunc(func(query string) (driver.Result, error) {
					return nil, errors.New("failed to connect")
				})
			})

			It("returns a successful sequence number", func() {
				seq, err := sequenceChecker.Check()
				Expect(err).ToNot(HaveOccurred())
				Expect(seq).To(ContainSubstring(expectedSeqNumber))
			})

			Context("and recover cmd returns -1", func() {
				BeforeEach(func() {
					mysqldCmd.RecoverSeqnoReturns("-1", nil)
				})

				It("returns an error", func() {
					_, err := sequenceChecker.Check()
					Expect(err).To(MatchError("Invalid sequence number -1"))
				})
			})

			Context("and recover cmd returns error", func() {
				BeforeEach(func() {
					mysqldCmd.RecoverSeqnoReturns("", errors.New("something went wrong"))
				})

				It("returns an unsuccessful Check", func() {
					_, err := sequenceChecker.Check()
					Expect(err).To(MatchError("something went wrong"))
				})
			})
		})
	})
})
