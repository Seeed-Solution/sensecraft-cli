package group

import (
	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

func NewGroupCmd() *cobra.Command {
	groupCmd := &cobra.Command{
		Use:   "group",
		Short: "Device group management",
	}
	groupCmd.AddCommand(newGroupListCmd())
	groupCmd.AddCommand(newGroupCreateCmd())
	groupCmd.AddCommand(newGroupUpdateCmd())
	groupCmd.AddCommand(newGroupDeleteCmd())
	groupCmd.AddCommand(newGroupMoveDevicesCmd())
	return groupCmd
}

func newGroupListCmd() *cobra.Command {
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all device groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetGroupService(ctx).ListGroups()
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	return cmd
}

func newGroupCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a device group (max 50 groups)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetGroupService(ctx).CreateGroup(name)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Group name (required)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newGroupUpdateCmd() *cobra.Command {
	var groupUUID, name string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a device group name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetGroupService(ctx).UpdateGroup(groupUUID, name)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&groupUUID, "group-uuid", "", "Device group UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "New group name (required)")
	_ = cmd.MarkFlagRequired("group-uuid")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newGroupDeleteCmd() *cobra.Command {
	var groupUUID string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a device group (devices move to default; L2: --yes)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetGroupService(ctx).DeleteGroup(groupUUID, yes)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&groupUUID, "group-uuid", "", "Device group UUID (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 delete")
	_ = cmd.MarkFlagRequired("group-uuid")
	return cmd
}

func newGroupMoveDevicesCmd() *cobra.Command {
	var groupUUID string
	var devices []string
	var yes bool
	cmd := &cobra.Command{
		Use:   "move-devices",
		Short: "Move devices to a specified group (L2: --yes)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetGroupService(ctx).MoveDevicesToGroup(groupUUID, devices, yes)
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&groupUUID, "group-uuid", "", "Target group UUID (required)")
	cmd.Flags().StringSliceVar(&devices, "eui", nil, "Device EUI; may be specified multiple times (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm L2 move")
	_ = cmd.MarkFlagRequired("group-uuid")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}
