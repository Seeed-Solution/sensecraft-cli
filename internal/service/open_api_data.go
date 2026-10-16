package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
	"sensecraft-cloud-terminal/internal/enum"
)

type DataService struct {
	client *client.OpenApiClient
}

func GetDataService(ctx app.CmdContext) DataService {
	return DataService{client: client.NewOpenApiClient(ctx.Config, ctx.SandBox)}
}

type TelemetryQuery struct {
	DeviceEUI     string
	ChannelIndex  string
	MeasurementID string
}

type ListTelemetryQuery struct {
	TelemetryQuery
	Limit     int
	TimeStart int64
	TimeEnd   int64
}

type AggregateChartQuery struct {
	TelemetryQuery
	Interval  string
	TimeStart int64
	TimeEnd   int64
}

func (s DataService) ViewLatestTelemetry(q TelemetryQuery) (json.RawMessage, error) {
	data, err := s.client.Get(enum.OpenApiUrlDataViewLatestTelemetry, telemetryValues(q))
	if err != nil {
		if errors.Is(err, client.ErrSandboxMode) {
			raw, _ := json.Marshal([]map[string]any{
				{
					"channel_index": 1,
					"points": []map[string]any{
						{
							"measurement_value": 25.5,
							"measurement_id":    "4097",
							"time":              "2020-03-26T07:28:28.575Z",
						},
					},
				},
			})
			return raw, nil
		}
		return nil, err
	}
	return data, nil
}

func (s DataService) ListTelemetry(q ListTelemetryQuery) (json.RawMessage, error) {
	values := telemetryValues(q.TelemetryQuery)
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	setTimeRange(values, q.TimeStart, q.TimeEnd)
	return s.client.Get(enum.OpenApiUrlDataListTelemetry, values)
}

func (s DataService) AggregateChartPoints(q AggregateChartQuery) (json.RawMessage, error) {
	values := telemetryValues(q.TelemetryQuery)
	if q.Interval != "" {
		values.Set("interval", q.Interval)
	}
	setTimeRange(values, q.TimeStart, q.TimeEnd)
	return s.client.Get(enum.OpenApiUrlDataAggregateChart, values)
}

func telemetryValues(q TelemetryQuery) url.Values {
	values := url.Values{}
	values.Set("device_eui", q.DeviceEUI)
	if q.ChannelIndex != "" {
		values.Set("channel_index", q.ChannelIndex)
	}
	if q.MeasurementID != "" {
		values.Set("measurement_id", q.MeasurementID)
	}
	return values
}

func setTimeRange(values url.Values, timeStart, timeEnd int64) {
	if timeStart > 0 {
		values.Set("time_start", strconv.FormatInt(timeStart, 10))
	}
	if timeEnd > 0 {
		values.Set("time_end", strconv.FormatInt(timeEnd, 10))
	}
}
