package store

// 默认分页参数，当调用方传入非法值（page <= 0 或 pageSize <= 0）时使用。
const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// paginate 计算分页切片的安全区间 [start, end]，并对非法的 page/pageSize 做规范化处理。
//
// 它是所有内存分页查询的统一入口，确保任意输入都不会导致切片越界 panic：
//   - page <= 0 时归一化为 1，pageSize <= 0 时归一化为默认值，超过上限时截断。
//   - start/end 始终被夹紧到 [0, total] 区间内。
//   - 当 start >= total（例如请求的页码远超实际数据页数）时返回空区间 (0, 0)，
//     由调用方据此返回空切片。
//
// 返回规范化后的 page/pageSize 以及夹紧后的 [start, end) 半开区间。
func paginate(page, pageSize, total int) (normPage, normPageSize, start, end int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	// start 由规范化后的 page 计算，保证 >= 0。
	start = (page - 1) * pageSize
	end = start + pageSize

	// 夹紧到 [0, total]：页码远超实际数据时 start >= total，返回空区间。
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	// 防御：保证 start <= end，避免 start > end 的切片越界。
	if start > end {
		start = end
	}

	return page, pageSize, start, end
}
