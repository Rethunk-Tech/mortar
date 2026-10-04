package cli

import (
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/doctor"
)

func liveDoctorReport(d control.Doctor, version string) doctor.Report {
	return doctor.LiveWithNativeHosts(doctor.Live{
		Version: d.Version, CommandVersion: version, DataDir: d.DataDir,
		Games: d.Games, Environment: d.Environment, NxmHandled: d.NxmHandled, NxmPrevious: d.NxmPrevious,
	})
}
