package util

import "testing"

func TestNewRingBuffer(t *testing.T) {
	t.Run("ZeroCapacity", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		NewRingBuffer(0)
	})

	t.Run("PositiveCapacity", func(t *testing.T) {
		buffer := NewRingBuffer(3)
		if buffer.capacity != 3 {
			t.Errorf("Expected %v, got %v", 3, buffer.capacity)
		}
		if buffer.size != 0 {
			t.Errorf("Expected size 0, got %v", buffer.size)
		}
		if buffer.head != 0 {
			t.Errorf("Expected head 0, got %v", buffer.head)
		}
		if buffer.lines == nil || len(buffer.lines) != 3 {
			t.Errorf("Expected lines slice of length 3, got %v", buffer.lines)
		}
	})
}

func TestAppend(t *testing.T) {
	t.Run("WithinCapacity", func(t *testing.T) {
		buffer := NewRingBuffer(3)
		buffer.Append("A")
		buffer.Append("B")
		if buffer.size != 2 {
			t.Errorf("Expected size 2, got %v", buffer.size)
		}
		if buffer.head != 2 {
			t.Errorf("Expected head 2, got %v", buffer.head)
		}
		expected := []string{"A", "B", ""}
		for i, v := range expected {
			if buffer.lines[i] != v {
				t.Errorf("At index %d, expected %v, got %v", i, v, buffer.lines[i])
			}
		}
	})

	t.Run("ExceedCapacity", func(t *testing.T) {
		buffer := NewRingBuffer(2)
		buffer.Append("A")
		buffer.Append("B")
		buffer.Append("C")
		if buffer.size != 2 {
			t.Errorf("Expected size 2, got %v", buffer.size)
		}
		if buffer.head != 1 {
			t.Errorf("Expected head 1, got %v", buffer.head)
		}
		expected := []string{"C", "B"}
		for i, v := range expected {
			if buffer.lines[i] != v {
				t.Errorf("At index %d, expected %v, got %v", i, v, buffer.lines[i])
			}
		}
	})
}

func TestGetRecent(t *testing.T) {
	t.Run("FromEmpty", func(t *testing.T) {
		buffer := NewRingBuffer(5)
		recent := buffer.Recent(3)
		if len(recent) != 0 {
			t.Errorf("Expected empty slice, got %v", recent)
		}
	})

	t.Run("LessThanCapacity", func(t *testing.T) {
		buffer := NewRingBuffer(5)
		buffer.Append("A")
		buffer.Append("B")
		recent := buffer.Recent(3)
		expected := []string{"A", "B"}
		if len(recent) != len(expected) {
			t.Errorf("Expected %v, got %v", expected, recent)
		}
		for i, v := range expected {
			if recent[i] != v {
				t.Errorf("At index %d, expected %v, got %v", i, v, recent[i])
			}
		}
	})

	t.Run("MoreThanCapacity", func(t *testing.T) {
		buffer := NewRingBuffer(5)
		buffer.Append("A")
		buffer.Append("B")
		buffer.Append("C")
		buffer.Append("D")
		buffer.Append("E")
		buffer.Append("F")
		recent := buffer.Recent(3)
		expected := []string{"D", "E", "F"}
		if len(recent) != len(expected) {
			t.Errorf("Expected %v, got %v", expected, recent)
		}
		for i, v := range expected {
			if recent[i] != v {
				t.Errorf("At index %d, expected %v, got %v", i, v, recent[i])
			}
		}
	})

	t.Run("RequestMoreThanSize", func(t *testing.T) {
		buffer := NewRingBuffer(5)
		buffer.Append("A")
		recent := buffer.Recent(10)
		expected := []string{"A"}
		if len(recent) != len(expected) {
			t.Errorf("Expected %v, got %v", expected, recent)
		}
		if recent[0] != "A" {
			t.Errorf("Expected A, got %v", recent[0])
		}
	})
}

func TestGetAll(t *testing.T) {
	buffer := NewRingBuffer(3)
	buffer.Append("A")
	buffer.Append("B")
	all := buffer.All()
	expected := []string{"A", "B"}
	if len(all) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, all)
	}
	for i, v := range expected {
		if all[i] != v {
			t.Errorf("At index %d, expected %v, got %v", i, v, all[i])
		}
	}
}
