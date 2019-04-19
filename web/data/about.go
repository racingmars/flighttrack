package data

type TableSize struct {
	TableName  string `db:"table_name"`
	SizeTable  string `db:"table_size"`
	SizeIndex  string `db:"indexes_size"`
	SizeTotal  string `db:"total_size"`
	ApproxRows int64  `db:"rowcount"`
}

func (d *DAO) GetTableSizes() ([]TableSize, error) {
	tables := make([]TableSize, 0)
	err := d.db.Select(&tables, "SELECT table_name, table_size, indexes_size, total_size, rowcount FROM dbsize ORDER BY raw_size DESC")
	return tables, err
}
