package data

import (
	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/service"

	"github.com/spf13/cobra"
)

func NewDataCmd() *cobra.Command {
	dataCmd := &cobra.Command{
		Use:   "data",
		Short: "Device sensor data",
	}
	dataCmd.AddCommand(newDataLatestCmd())
	dataCmd.AddCommand(newDataListCmd())
	dataCmd.AddCommand(newDataChartCmd())
	return dataCmd
}

func newDataLatestCmd() *cobra.Command {
	var deviceEUI, measurementID, channelIndex string
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "latest",
		Short: "Get the latest telemetry data for a device within the past year",
		Long: `Returns the latest telemetry points within one year.
If channel_index is omitted, returns the latest point for each channel.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			ctx.SandBox = sandbox
			data, err := service.GetDataService(ctx).ViewLatestTelemetry(service.TelemetryQuery{
				DeviceEUI:     deviceEUI,
				ChannelIndex:  channelIndex,
				MeasurementID: measurementID,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&deviceEUI, "eui", "", "Device EUI (required)")
	cmd.Flags().StringVar(&measurementID, "measurement-id", "", "Measurement ID")
	cmd.Flags().StringVar(&channelIndex, "channel-index", "", "Channel index")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "Sandbox mode: no real network; return sample JSON")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}

func newDataListCmd() *cobra.Command {
	var (
		deviceEUI, measurementID, channelIndex string
		limit                                  int
		timeStart, timeEnd                     int64
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List historical telemetry data for a device",
		Long: `Returns historical telemetry for a device.
At most about one month of data per request; only data within the last three months can be queried.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetDataService(ctx).ListTelemetry(service.ListTelemetryQuery{
				TelemetryQuery: service.TelemetryQuery{
					DeviceEUI:     deviceEUI,
					ChannelIndex:  channelIndex,
					MeasurementID: measurementID,
				},
				Limit:     limit,
				TimeStart: timeStart,
				TimeEnd:   timeEnd,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&deviceEUI, "eui", "", "Device EUI (required)")
	cmd.Flags().StringVar(&measurementID, "measurement-id", "", "Measurement ID")
	cmd.Flags().StringVar(&channelIndex, "channel-index", "", "Channel index")
	cmd.Flags().IntVar(&limit, "limit", 0, "Number of records to return")
	cmd.Flags().Int64Var(&timeStart, "time-start", 0, "Start timestamp (milliseconds)")
	cmd.Flags().Int64Var(&timeEnd, "time-end", 0, "End timestamp (milliseconds)")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}

func newDataChartCmd() *cobra.Command {
	var (
		deviceEUI, measurementID, channelIndex string
		interval                               string
		timeStart, timeEnd                     int64
	)
	cmd := &cobra.Command{
		Use:   "chart",
		Short: "Get aggregated telemetry chart points for a device",
		Long: `Splits a large time range into intervals and returns the average value of each interval.
Up to one year of data; at most 250 points per measurement (intervals are auto-resized if needed).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := app.GetCmdContext()
			if err != nil {
				return err
			}
			data, err := service.GetDataService(ctx).AggregateChartPoints(service.AggregateChartQuery{
				TelemetryQuery: service.TelemetryQuery{
					DeviceEUI:     deviceEUI,
					ChannelIndex:  channelIndex,
					MeasurementID: measurementID,
				},
				Interval:  interval,
				TimeStart: timeStart,
				TimeEnd:   timeEnd,
			})
			if err != nil {
				return err
			}
			return ctx.PrintRawData(data)
		},
	}
	cmd.Flags().StringVar(&deviceEUI, "eui", "", "Device EUI (required)")
	cmd.Flags().StringVar(&measurementID, "measurement-id", "", "Measurement ID")
	cmd.Flags().StringVar(&channelIndex, "channel-index", "", "Channel index")
	cmd.Flags().StringVar(&interval, "interval", "60", "Interval length in minutes (default 60)")
	cmd.Flags().Int64Var(&timeStart, "time-start", 0, "Start timestamp (milliseconds)")
	cmd.Flags().Int64Var(&timeEnd, "time-end", 0, "End timestamp (milliseconds)")
	_ = cmd.MarkFlagRequired("eui")
	return cmd
}
