package commandui

import (
	"testing"

	"github.com/parfenovvs/lazylogcat/internal/model"
)

func TestDeviceSingleSelectItems(t *testing.T) {
	t.Run("EmptySlice", func(t *testing.T) {
		got := deviceSingleSelectItems(nil)
		if got != nil {
			t.Errorf("deviceSingleSelectItems(nil) = %v, want nil", got)
		}
	})

	t.Run("ShortName_NoTruncation", func(t *testing.T) {
		devices := []model.Device{
			{Id: "abc123", Name: "Pixel"},
		}
		got := deviceSingleSelectItems(devices)
		if len(got) != 1 {
			t.Fatalf("got %d items, want 1", len(got))
		}
		if got[0].Key != "abc123" {
			t.Errorf("Key = %q, want %q", got[0].Key, "abc123")
		}
		if got[0].Columns[0] != "Pixel" {
			t.Errorf("Name column = %q, want %q", got[0].Columns[0], "Pixel")
		}
		if got[0].Columns[1] != "abc123" {
			t.Errorf("ID column = %q, want %q", got[0].Columns[1], "abc123")
		}
	})

	t.Run("LongName_Truncated", func(t *testing.T) {
		devices := []model.Device{
			{Id: "dev1", Name: "VeryLongDeviceNameThatExceeds18"},
		}
		got := deviceSingleSelectItems(devices)
		if len(got) != 1 {
			t.Fatalf("got %d items, want 1", len(got))
		}
		want := "VeryLongDeviceN..."
		if got[0].Columns[0] != want {
			t.Errorf("truncated name = %q, want %q", got[0].Columns[0], want)
		}
	})

	t.Run("ExactlyMaxLen_NoTruncation", func(t *testing.T) {
		// 18 chars exactly
		devices := []model.Device{
			{Id: "dev2", Name: "123456789012345678"},
		}
		got := deviceSingleSelectItems(devices)
		if len(got) != 1 {
			t.Fatalf("got %d items, want 1", len(got))
		}
		if got[0].Columns[0] != "123456789012345678" {
			t.Errorf("name = %q, want no truncation", got[0].Columns[0])
		}
	})

	t.Run("MultipleDevices_OrderPreserved", func(t *testing.T) {
		devices := []model.Device{
			{Id: "id1", Name: "First"},
			{Id: "id2", Name: "Second"},
			{Id: "id3", Name: "Third"},
		}
		got := deviceSingleSelectItems(devices)
		if len(got) != 3 {
			t.Fatalf("got %d items, want 3", len(got))
		}
		for i, d := range devices {
			if got[i].Key != d.Id {
				t.Errorf("item[%d].Key = %q, want %q", i, got[i].Key, d.Id)
			}
			if got[i].Columns[0] != d.Name {
				t.Errorf("item[%d].Name = %q, want %q", i, got[i].Columns[0], d.Name)
			}
		}
	})
}
