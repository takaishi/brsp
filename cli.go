package brsp

import (
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

var Version = "dev"
var Revision = "HEAD"

type GlobalOptions struct {
}

type CLI struct {
	GenerateDataKey   *GenerateDataKeyCommandOption   `cmd:"generate-data-key" help:""`
	BackupParameters  *BackupParametersCommandOption  `cmd:"backup-parameters" help:""`
	BackupSecrets     *BackupSecretsCommandOption     `cmd:"backup-secrets" help:""`
	DownloadBackup    *DownloadBackupCommandOption    `cmd:"download-backup" help:""`
	RestoreSecrets    *RestoreSecretsCommandOption    `cmd:"restore-secrets" help:""`
	RestoreParameters *RestoreParametersCommandOption `cmd:"restore-parameters" help:""`
	Version           VersionFlag                     `name:"version" help:"show version"`
}

type VersionFlag string

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Printf("%s-%s\n", Version, Revision)
	app.Exit(0)
	return nil
}

func RunCLI(ctx context.Context, args []string) error {
	var cli CLI
	parser, err := kong.New(&cli)
	if err != nil {
		return err
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	cmd := strings.Fields(kctx.Command())[0]
	if cmd == "version" {
		fmt.Println(Version)
		return nil
	}

	app := New(&cli)
	return app.Dispatch(ctx, cmd)
}

func (a *App) Dispatch(ctx context.Context, command string) error {
	switch command {
	case "generate-data-key":
		cmd, err := NewGenerateDataKeyCommand(a.CLI.GenerateDataKey)
		if err != nil {
			return err
		}
		return cmd.Run()
	case "backup-parameters":
		cmd, err := NewBackupParametersCommand(a.CLI.BackupParameters)
		if err != nil {
			return err
		}
		return cmd.Run()
	case "backup-secrets":
		cmd, err := NewBackupSecretsCommand(a.CLI.BackupSecrets)
		if err != nil {
			return err
		}
		return cmd.Run()
	case "download-backup":
		cmd, err := NewDownloadBackupCommand(a.CLI.DownloadBackup)
		if err != nil {
			return err
		}
		return cmd.Run()
	case "restore-secrets":
		cmd, err := NewRestoreSecretsCommand(a.CLI.RestoreSecrets)
		if err != nil {
			return err
		}
		return cmd.Run()
	case "restore-parameters":
		cmd, err := NewRestoreParametersCommand(a.CLI.RestoreParameters)
		if err != nil {
			return err
		}
		return cmd.Run()

	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func getAwsConfig() (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO())
}

func getTargetAwsConfig(targetRegion string) (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO(), config.WithRegion(targetRegion))
}
