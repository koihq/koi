export interface BaseListParams {
  page?: number;
  pageSize?: number;
}

export interface ApiListResponse<T> {
  items: T[];
  totalItems: number;
  totalPages: number;
  currentPage: number;
  hasNext: boolean;
  hasPrev: boolean;
}
