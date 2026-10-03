package settings

import "slices"

func fillPortable(s Settings) Portable {
	return Portable{
		Version:                      exportVersion,
		Language:                     s.Language,
		Accent:                       s.Accent,
		Background:                   s.Background,
		LastGame:                     s.LastGame,
		BackupsKept:                  s.BackupsKept,
		ListColumns:                  slices.Clone(s.ListColumns),
		ListSortColumn:               s.ListSortColumn,
		ListSortDir:                  s.ListSortDir,
		ListGroupBy:                  s.ListGroupBy,
		CheckModUpdatesOnStart:       s.CheckModUpdatesOnStart,
		TellWhenSmapiOut:             s.TellWhenSmapiOut,
		KeepInTray:                   s.KeepInTray,
		IncludeBetaReleases:          s.IncludeBetaReleases,
		IncludePrereleaseModVersions: s.IncludePrereleaseModVersions,
		CheckOnlyEnabledMods:         s.CheckOnlyEnabledMods,
		EnableModsWhenInstalled:      s.EnableModsWhenInstalled,
		TipsSeen:                     slices.Clone(s.TipsSeen),
		NexusPreferredDownloadServer: s.NexusPreferredDownloadServer,
		NxmRedirectOtherGames:        s.NxmRedirectOtherGames,
		OnPlay:                       s.OnPlay,
		BackupBeforePlay:             s.BackupBeforePlay,
		LaunchBackupsKept:            s.LaunchBackupsKept,
		UpdateModsBeforePlayDefault:  s.UpdateModsBeforePlayDefault,
		RunsKept:                     s.RunsKept,
		ConsoleLogCap:                s.ConsoleLogCap,
		ParallelDownloads:            s.ParallelDownloads,
		UpdateCheckIntervalMinutes:   s.UpdateCheckIntervalMinutes,
		NotifyModUpdates:             s.NotifyModUpdates,
		KeepDownloadArchives:         s.KeepDownloadArchives,
		StoreRetentionDays:           s.StoreRetentionDays,
		NxmDefaultProfile:            s.NxmDefaultProfile,
		DefaultModsView:              s.DefaultModsView,
		ConfirmRemovals:              s.ConfirmRemovals,
		CosmeticConflicts:            s.CosmeticConflicts,
		BackgroundBadgeChecks:        s.BackgroundBadgeChecks,
		StartScreen:                  s.StartScreen,
		Dates:                        s.Dates,
		TrashRetentionDays:           s.TrashRetentionDays,
		HistoryEventsKept:            s.HistoryEventsKept,
		NotifyDownloadFinished:       s.NotifyDownloadFinished,
		NotifyDownloadFailed:         s.NotifyDownloadFailed,
		NotifyRunCrashed:             s.NotifyRunCrashed,
		Density:                      s.Density,
		GridCardSize:                 s.GridCardSize,
		ShowAuthorOnCards:            s.ShowAuthorOnCards,
		ReduceMotion:                 s.ReduceMotion,
		ProfileHero:                  s.ProfileHero,
		EnableRequirements:           s.EnableRequirements,
		MissingRequirements:          s.MissingRequirements,
		ReuseFomodChoices:            s.ReuseFomodChoices,
		DriftChecks:                  s.DriftChecks,
		SmapiBuilds:                  s.SmapiBuilds,
		AutoInstallMortarUpdates:     s.AutoInstallMortarUpdates,
		AutoTrackNexus:               s.AutoTrackNexus,
		DefaultLaunchMethod:          s.DefaultLaunchMethod,
		ShowSmapiConsole:             s.ShowSmapiConsole,
		ConsoleLevel:                 s.ConsoleLevel,
		ConsoleTimestamps:            s.ConsoleTimestamps,
		ConsoleFollow:                s.ConsoleFollow,
		LanName:                      s.LanName,
		LanAutoAcceptSameAccount:     s.LanAutoAcceptSameAccount,
		DownloadFolder:               s.DownloadFolder,
	}
}

func copyPortable(dst *Settings, p Portable, present map[string]struct{}) {
	has := func(k string) bool { _, ok := present[k]; return ok }
	if has("language") {
		dst.Language = p.Language
	}
	if has("accent") {
		dst.Accent = p.Accent
	}
	if has("background") {
		dst.Background = p.Background
	}
	if has("lastGame") {
		dst.LastGame = p.LastGame
	}
	if has("backupsKept") {
		dst.BackupsKept = p.BackupsKept
	}
	if has("listColumns") {
		dst.ListColumns = slices.Clone(p.ListColumns)
	}
	if has("listSortColumn") {
		dst.ListSortColumn = p.ListSortColumn
	}
	if has("listSortDir") {
		dst.ListSortDir = p.ListSortDir
	}
	if has("listGroupBy") {
		dst.ListGroupBy = p.ListGroupBy
	}
	if has("checkModUpdatesOnStart") {
		dst.CheckModUpdatesOnStart = p.CheckModUpdatesOnStart
	}
	if has("tellWhenSmapiOut") {
		dst.TellWhenSmapiOut = p.TellWhenSmapiOut
	}
	if has("keepInTray") {
		dst.KeepInTray = p.KeepInTray
	}
	if has("includeBetaReleases") {
		dst.IncludeBetaReleases = p.IncludeBetaReleases
	}
	if has("includePrereleaseModVersions") {
		dst.IncludePrereleaseModVersions = p.IncludePrereleaseModVersions
	}
	if has("checkOnlyEnabledMods") {
		dst.CheckOnlyEnabledMods = p.CheckOnlyEnabledMods
	}
	if has("enableModsWhenInstalled") {
		dst.EnableModsWhenInstalled = p.EnableModsWhenInstalled
	}
	if has("tipsSeen") {
		dst.TipsSeen = slices.Clone(p.TipsSeen)
	}
	if has("nexusPreferredDownloadServer") {
		dst.NexusPreferredDownloadServer = p.NexusPreferredDownloadServer
	}
	if has("nxmRedirectOtherGames") {
		dst.NxmRedirectOtherGames = p.NxmRedirectOtherGames
	}
	if has("onPlay") {
		dst.OnPlay = p.OnPlay
	}
	if has("backupBeforePlay") {
		dst.BackupBeforePlay = p.BackupBeforePlay
	}
	if has("launchBackupsKept") {
		dst.LaunchBackupsKept = p.LaunchBackupsKept
	}
	if has("updateModsBeforePlayDefault") {
		dst.UpdateModsBeforePlayDefault = p.UpdateModsBeforePlayDefault
	}
	if has("runsKept") {
		dst.RunsKept = p.RunsKept
	}
	if has("consoleLogCap") {
		dst.ConsoleLogCap = p.ConsoleLogCap
	}
	if has("parallelDownloads") {
		dst.ParallelDownloads = p.ParallelDownloads
	}
	if has("updateCheckIntervalMinutes") {
		dst.UpdateCheckIntervalMinutes = p.UpdateCheckIntervalMinutes
	}
	if has("notifyModUpdates") {
		dst.NotifyModUpdates = p.NotifyModUpdates
	}
	if has("keepDownloadArchives") {
		dst.KeepDownloadArchives = p.KeepDownloadArchives
	}
	if has("storeRetentionDays") {
		dst.StoreRetentionDays = p.StoreRetentionDays
	}
	if has("nxmDefaultProfile") {
		dst.NxmDefaultProfile = p.NxmDefaultProfile
	}
	if has("defaultModsView") {
		dst.DefaultModsView = p.DefaultModsView
	}
	if has("confirmRemovals") {
		dst.ConfirmRemovals = p.ConfirmRemovals
	}
	if has("cosmeticConflicts") {
		dst.CosmeticConflicts = p.CosmeticConflicts
	}
	if has("backgroundBadgeChecks") {
		dst.BackgroundBadgeChecks = p.BackgroundBadgeChecks
	}
	if has("startScreen") {
		dst.StartScreen = p.StartScreen
	}
	if has("dates") {
		dst.Dates = p.Dates
	}
	if has("trashRetentionDays") {
		dst.TrashRetentionDays = p.TrashRetentionDays
	}
	if has("historyEventsKept") {
		dst.HistoryEventsKept = p.HistoryEventsKept
	}
	if has("notifyDownloadFinished") {
		dst.NotifyDownloadFinished = p.NotifyDownloadFinished
	}
	if has("notifyDownloadFailed") {
		dst.NotifyDownloadFailed = p.NotifyDownloadFailed
	}
	if has("notifyRunCrashed") {
		dst.NotifyRunCrashed = p.NotifyRunCrashed
	}
	if has("density") {
		dst.Density = p.Density
	}
	if has("gridCardSize") {
		dst.GridCardSize = p.GridCardSize
	}
	if has("showAuthorOnCards") {
		dst.ShowAuthorOnCards = p.ShowAuthorOnCards
	}
	if has("reduceMotion") {
		dst.ReduceMotion = p.ReduceMotion
	}
	if has("profileHero") {
		dst.ProfileHero = p.ProfileHero
	}
	if has("enableRequirements") {
		dst.EnableRequirements = p.EnableRequirements
	}
	if has("missingRequirements") {
		dst.MissingRequirements = p.MissingRequirements
	}
	if has("reuseFomodChoices") {
		dst.ReuseFomodChoices = p.ReuseFomodChoices
	}
	if has("driftChecks") {
		dst.DriftChecks = p.DriftChecks
	}
	if has("smapiBuilds") {
		dst.SmapiBuilds = p.SmapiBuilds
	}
	if has("autoInstallMortarUpdates") {
		dst.AutoInstallMortarUpdates = p.AutoInstallMortarUpdates
	}
	if has("autoTrackNexus") {
		dst.AutoTrackNexus = p.AutoTrackNexus
	}
	if has("defaultLaunchMethod") {
		dst.DefaultLaunchMethod = p.DefaultLaunchMethod
	}
	if has("showSmapiConsole") {
		dst.ShowSmapiConsole = p.ShowSmapiConsole
	}
	if has("consoleLevel") {
		dst.ConsoleLevel = p.ConsoleLevel
	}
	if has("consoleTimestamps") {
		dst.ConsoleTimestamps = p.ConsoleTimestamps
	}
	if has("consoleFollow") {
		dst.ConsoleFollow = p.ConsoleFollow
	}
	if has("lanName") {
		dst.LanName = p.LanName
	}
	if has("lanAutoAcceptSameAccount") {
		dst.LanAutoAcceptSameAccount = p.LanAutoAcceptSameAccount
	}
	if has("downloadFolder") {
		dst.DownloadFolder = p.DownloadFolder
	}
}
