package metering

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

var logger = logrus.StandardLogger().WithField("metering", true)

func RecordLogin(loginType string, userID, instanceID int64) {
	logger.WithFields(logrus.Fields{
		"action":       "login",
		"login_method": loginType,
		"instance_id":  fmt.Sprintf("%d", instanceID),
		"user_id":      fmt.Sprintf("%d", userID),
	}).Info("Login")
}
