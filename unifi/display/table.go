package display

import (
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/johnweldon/unifi-scheduler/unifi"
)

type Renderer interface {
	Render() string
}

func ClientsTable(out io.Writer, clients []unifi.Client) Renderer {
	configs := []table.ColumnConfig{
		{Name: "Name", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
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
		{Name: "Name", Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Event"},
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

	getName := func(mac unifi.MAC) string {
		name, _ := displayName(mac)
		return name
	}

	for _, event := range events {
		var (
			name string
			ok   bool
		)

		for _, mac := range []unifi.MAC{
			event.User,
			event.Client,
			event.Guest,
		} {
			if name, ok = displayName(mac); ok {
				evt := event.Key[7:]

				to := "-"
				from := "-"
				switch event.Key {
				case unifi.EventTypeWirelessUserRoam:
					to = getName(event.AccessPointTo)
					from = getName(event.AccessPointFrom)
				case
					unifi.EventTypeWirelessGuestDisconnected,
					unifi.EventTypeWirelessUserDisconnected:
					from = getName(event.AccessPoint)
				case unifi.EventTypeWirelessUserConnected:
					to = getName(event.AccessPoint)
				case unifi.EventTypeWirelessUserRoamRadio:
					from = fmt.Sprintf("%s (%d)", event.RadioFrom, event.ChannelFrom)
					to = fmt.Sprintf("%s (%d)", event.RadioTo, event.ChannelTo)
				}

				t.AppendRow([]interface{}{
					name, evt, from, to, event.TimeStamp.ShortTime(), event.TimeStamp.String(),
				})
			}
		}
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
