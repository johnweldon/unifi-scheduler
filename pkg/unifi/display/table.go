package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
)

type Renderer interface {
	Render() string
}

func ClientsTable(out io.Writer, clients []unifi.Client) Renderer {
	configs := []table.ColumnConfig{
		{Name: "Name", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight, WidthMax: 25},
		{Name: "B"},
		{Name: "G"},
		{Name: "W"},
		{Name: "IP"},
		{Name: "Associated"},
		{Name: "Rx", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Tx", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Rx Rate", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Tx Rate", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Link"},
	}

	headerRow := table.Row{}
	for _, c := range configs {
		headerRow = append(headerRow, c.Name)
	}

	t := table.NewWriter()
	t.SetStyle(StyleDefault)
	t.SetColumnConfigs(configs)
	t.SetOutputMirror(out)

	t.AppendHeader(headerRow)
	for _, client := range clients {
		t.AppendRow([]interface{}{
			client.DisplayName(),
			string(client.IsBlockedGlyph()),
			string(client.IsGuestGlyph()),
			string(client.IsWiredGlyph()),
			client.DisplayIP(),
			client.DisplayLastAssociated(),
			client.DisplayReceivedBytes(),
			client.DisplaySentBytes(),
			client.DisplayReceiveRate(),
			client.DisplaySendRate(),
			client.DisplaySwitchName(),
		})
	}
	t.AppendFooter(table.Row{fmt.Sprintf("Total %d", t.Length())})
	return t
}

func EventsTable(out io.Writer, displayName func(unifi.MAC) (string, bool), events []unifi.Event) Renderer {
	configs := []table.ColumnConfig{
		{Name: "Name", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight, WidthMax: 25},
		{Name: "Event", WidthMax: 15},
		{Name: "From"},
		{Name: "To"},
		{Name: "When"},
		{Name: "Ago"},
	}

	headerRow := table.Row{}
	for _, c := range configs {
		headerRow = append(headerRow, c.Name)
	}

	t := table.NewWriter()
	t.SetStyle(StyleDefault)
	t.SetColumnConfigs(configs)
	t.SetOutputMirror(out)

	t.AppendHeader(headerRow)

	getName := func(macs ...unifi.MAC) string {
		var names []string
		for _, mac := range macs {
			names = append(names, string(mac))

			if name, ok := displayName(mac); ok {
				return name
			}
		}

		return strings.Join(names, ",")
	}

	for _, event := range events {
		name := getName(event.MAC)
		evt := event.Key[7:]
		to := "-"
		from := "-"

		switch event.Key {

		case unifi.EventTypeWirelessUserRoam:

			to = getName(event.AccessPointTo)
			from = getName(event.AccessPointFrom)
			name = getName(event.User)

		case unifi.EventTypeWirelessGuestDisconnected:

			from = getName(event.AccessPoint)
			name = getName(event.Guest)

		case unifi.EventTypeWirelessUserDisconnected:

			from = getName(event.AccessPoint)
			name = getName(event.User)

		case unifi.EventTypeWirelessUserConnected:

			to = getName(event.AccessPoint)
			name = getName(event.User)

		case unifi.EventTypeWirelessUserRoamRadio:

			from = fmt.Sprintf("%s (%d)", event.RadioFrom, event.ChannelFrom)
			to = fmt.Sprintf("%s (%d)", event.RadioTo, event.ChannelTo)
			name = getName(event.User)

		case unifi.EventTypeLANUserConnected:

			to = getName(event.Switch)
			name = getName(event.User)

		case unifi.EventTypeLANUserDisconnected:

			from = getName(event.Switch)
			name = getName(event.User)

		case unifi.EventTypeLANGuestConnected:

			to = getName(event.Switch)
			name = getName(event.Guest)

		case
			unifi.EventTypeLANClientBlocked,
			unifi.EventTypeLANClientUnblocked,
			unifi.EventTypeWirelessClientBlocked,
			unifi.EventTypeWirelessClientUnblocked:

			name = getName(event.Client)

		case
			unifi.EventTypeAccessPointAdopted,
			unifi.EventTypeAccessPointAutoReadopted,
			unifi.EventTypeAccessPointChannelChanged,
			unifi.EventTypeAccessPointConnected,
			unifi.EventTypeAccessPointDeleted,
			unifi.EventTypeAccessPointDetectRogueAP,
			unifi.EventTypeAccessPointIsolated,
			unifi.EventTypeAccessPointIsolated,
			unifi.EventTypeAccessPointLostContact,
			unifi.EventTypeAccessPointPossibleInterference,
			unifi.EventTypeAccessPointRestarted,
			unifi.EventTypeAccessPointRestartedUnknown,
			unifi.EventTypeAccessPointUpgradeFailed,
			unifi.EventTypeAccessPointUpgradeScheduled,
			unifi.EventTypeAccessPointUpgraded:

			name = getName(event.AccessPoint)

		case
			unifi.EventTypeBridgeAutoReadopted,
			unifi.EventTypeBridgeChannelChanged,
			unifi.EventTypeBridgeConnected,
			unifi.EventTypeBridgeLinkRadioChanged,
			unifi.EventTypeBridgeLostContact,
			unifi.EventTypeBridgeRestarted,
			unifi.EventTypeBridgeRestartedUnknown,
			unifi.EventTypeBridgeUpgradeFailed,
			unifi.EventTypeBridgeUpgradeScheduled,
			unifi.EventTypeBridgeUpgraded:

			name = getName(event.Bridge)

		case
			unifi.EventTypeDMConnected,
			unifi.EventTypeDMUpgraded:

			name = getName(event.DM)

		case unifi.EventTypeGatewayWANTransition:

			name = getName(event.Gateway)

		case
			unifi.EventTypeSwitchAutoReadopted,
			unifi.EventTypeSwitchConnected,
			unifi.EventTypeSwitchDetectRogueDHCP,
			unifi.EventTypeSwitchFirmwareCheckFailed,
			unifi.EventTypeSwitchFirmwareDownloadFailed,
			unifi.EventTypeSwitchLostContact,
			unifi.EventTypeSwitchPOEDisconnect,
			unifi.EventTypeSwitchRestarted,
			unifi.EventTypeSwitchRestartedUnknown,
			unifi.EventTypeSwitchSTPPortBlocking,
			unifi.EventTypeSwitchUpgradeFailed,
			unifi.EventTypeSwitchUpgradeScheduled,
			unifi.EventTypeSwitchUpgraded:

			name = getName(event.Switch)

		}

		if name == "" {
			name = string(event.Key)
		}

		t.AppendRow([]interface{}{
			name, evt, from, to, event.TimeStamp.ShortTime(), event.TimeStamp.String(),
		})
	}

	t.AppendFooter(table.Row{fmt.Sprintf("Total %d", t.Length())})

	return t
}

var StyleDefault = table.Style{
	Name:    "StyleDefault",
	Box:     table.StyleBoxDefault,
	Color:   table.ColorOptionsDefault,
	Format:  table.FormatOptionsDefault,
	HTML:    table.DefaultHTMLOptions,
	Options: table.OptionsNoBordersAndSeparators,
	Title:   table.TitleOptionsBright,
}
