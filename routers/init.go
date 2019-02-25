// Copyright 2016 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package routers

import (
	"path"
	"strings"

	"code.gitea.io/git"
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/migrations"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/cron"
	"code.gitea.io/gitea/modules/highlight"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/mailer"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/ssh"

	macaron "gopkg.in/macaron.v1"
)

func checkRunMode() {
	switch setting.Cfg.Section("").Key("RUN_MODE").String() {
	case "prod":
		macaron.Env = macaron.PROD
		macaron.ColorLog = false
		setting.ProdMode = true
	default:
		git.Debug = true
	}
	log.Info("Run Mode: %s", strings.Title(macaron.Env))
}

// NewServices init new services
func NewServices() {
	setting.NewServices()
	mailer.NewContext()
	cache.NewContext()
}

// GlobalInit is for global configuration reload-able.
func GlobalInit() {
	setting.NewContext()
	setting.CheckLFSVersion()
	log.Trace("AppPath: %s", setting.AppPath)
	log.Trace("AppWorkPath: %s", setting.AppWorkPath)
	log.Trace("Custom path: %s", setting.CustomPath)
	log.Trace("Log path: %s", setting.LogRootPath)
	models.LoadConfigs()
	NewServices()

	if setting.InstallLock {
		highlight.NewContext()
		markup.Init()

		if err := models.NewEngine(migrations.Migrate); err != nil {
			log.Fatal(4, "Failed to initialize ORM engine: %v", err)
		}
		models.HasEngine = true
		if err := models.InitOAuth2(); err != nil {
			log.Fatal(4, "Failed to initialize OAuth2 support: %v", err)
		}

		models.LoadRepoConfig()
		models.NewRepoContext()

		// Booting long running goroutines.
		cron.NewContext()
		models.InitIssueIndexer()
		models.InitRepoIndexer()
		models.InitSyncMirrors()
		models.InitDeliverHooks()
		models.InitTestPullRequests()
		log.NewGitLogger(path.Join(setting.LogRootPath, "http.log"))
	}
	if models.EnableSQLite3 {
		log.Info("SQLite3 Supported")
	}
	if models.EnableTiDB {
		log.Info("TiDB Supported")
	}
	checkRunMode()

	if setting.InstallLock && setting.SSH.StartBuiltinServer {
		ssh.Listen(setting.SSH.ListenHost, setting.SSH.ListenPort, setting.SSH.ServerCiphers, setting.SSH.ServerKeyExchanges, setting.SSH.ServerMACs)
		log.Info("SSH server started on %s:%d. Cipher list (%v), key exchange algorithms (%v), MACs (%v)", setting.SSH.ListenHost, setting.SSH.ListenPort, setting.SSH.ServerCiphers, setting.SSH.ServerKeyExchanges, setting.SSH.ServerMACs)
	}
}