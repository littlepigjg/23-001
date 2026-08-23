package store

import "testing"

// TestPaginate 验证分页索引计算在各类非法输入下都不会产生越界区间。
func TestPaginate(t *testing.T) {
	tests := []struct {
		name             string
		page, pageSize   int
		total            int
		wantPage         int
		wantPageSize     int
		wantStart, wantEnd int
	}{
		{"合法第一页", 1, 20, 5, 1, 20, 0, 5},
		{"合法第二页部分", 2, 3, 5, 2, 3, 3, 5},
		{"page=0 归一化", 0, 20, 5, 1, 20, 0, 5},
		{"page 为负", -1, 20, 5, 1, 20, 0, 5},
		{"页码远超数据 (page=100)", 100, 20, 3, 100, 20, 3, 3}, // start 夹紧到 total -> 空区间
		{"pageSize=0 归一化", 1, 0, 5, 1, 20, 0, 5},
		{"pageSize 为负", 1, -5, 5, 1, 20, 0, 5},
		{"pageSize 超上限", 1, 5000, 3, 1, 100, 0, 3},
		{"空数据集", 5, 20, 0, 5, 20, 0, 0},
		{"空数据集 page=0", 0, 20, 0, 1, 20, 0, 0},
		{"刚好一页满", 1, 5, 5, 1, 5, 0, 5},
		{"第二页为空但合法", 2, 5, 5, 2, 5, 5, 5}, // start==total==end -> 空切片，不越界
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPage, gotPageSize, gotStart, gotEnd := paginate(tt.page, tt.pageSize, tt.total)
			if gotPage != tt.wantPage || gotPageSize != tt.wantPageSize ||
				gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("paginate(%d, %d, %d) = (page=%d, pageSize=%d, start=%d, end=%d); want (%d, %d, %d, %d)",
					tt.page, tt.pageSize, tt.total,
					gotPage, gotPageSize, gotStart, gotEnd,
					tt.wantPage, tt.wantPageSize, tt.wantStart, tt.wantEnd)
			}
			if gotStart < 0 || gotEnd < 0 || gotStart > gotEnd || gotStart > tt.total || gotEnd > tt.total {
				t.Errorf("非法区间 [start=%d, end=%d] (total=%d)", gotStart, gotEnd, tt.total)
			}
		})
	}
}
