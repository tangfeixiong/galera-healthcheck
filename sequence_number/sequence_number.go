package sequence_number

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/mysqld_cmd"
	"github.com/pivotal-golang/lager"
	"strconv"
)

type SequenceNumberchecker struct {
	db        *sql.DB
	config    config.Config
	logger    lager.Logger
	mysqldCmd mysqld_cmd.MysqldCmd
}

func New(db *sql.DB, mysqldCmd mysqld_cmd.MysqldCmd, config config.Config, logger lager.Logger) *SequenceNumberchecker {
	return &SequenceNumberchecker{
		db:        db,
		config:    config,
		logger:    logger,
		mysqldCmd: mysqldCmd,
	}
}

func (s *SequenceNumberchecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	seqno, err := s.check()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errMsg := fmt.Sprintf("Failed to determine sequence number: %s", err.Error())
		s.logger.Error(errMsg, err)
		w.Write([]byte(errMsg))
		return
	}

	s.logger.Debug(fmt.Sprintf("Response body: %s", seqno))
	w.Write([]byte(seqno))
}

func (s *SequenceNumberchecker) check() (string, error) {
	s.logger.Info("Checking sequence number of mariadb node...")

	if s.dbReachable() {
		return "", errors.New("can't determine sequence number when database is running")
	} else {
		returnedSeqNo, err := s.readSeqNoFromRecoverCmd()
		if err != nil {
			return "", err
		}

		returnedSeqNoInt, converr := strconv.Atoi(returnedSeqNo)
		if converr != nil {
			return "", converr
		}

		if returnedSeqNoInt < 0 {
			return "", errors.New(fmt.Sprintf("Invalid sequence number %s", returnedSeqNo))
		}

		return returnedSeqNo, nil
	}
}

func (s *SequenceNumberchecker) readSeqNoFromRecoverCmd() (string, error) {
	s.logger.Info("Reading seqno from logs")
	seqno, err := s.mysqldCmd.RecoverSeqno()
	if err != nil {
		s.logger.Error("Failed to retrieve seqno from logs", err)
		return "", err
	}

	return seqno, nil
}

func (s *SequenceNumberchecker) dbReachable() bool {
	_, err := s.db.Exec("SHOW VARIABLES")
	if err != nil {
		s.logger.Info(fmt.Sprintf("Database not reachable, continuing: %s", err.Error()))
	}
	return err == nil
}