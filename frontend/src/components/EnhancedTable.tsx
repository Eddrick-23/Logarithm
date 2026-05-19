import * as React from "react";
import { alpha } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TablePagination from "@mui/material/TablePagination";
import TableRow from "@mui/material/TableRow";
import TableSortLabel from "@mui/material/TableSortLabel";
import Toolbar from "@mui/material/Toolbar";
import Typography from "@mui/material/Typography";
import Paper from "@mui/material/Paper";
import Checkbox from "@mui/material/Checkbox";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import DeleteIcon from "@mui/icons-material/Delete";
import FilterListIcon from "@mui/icons-material/FilterList";
import { visuallyHidden } from "@mui/utils";
import { InputAdornment, Stack, TextField } from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";

function descendingComparator<T>(a: T, b: T, orderBy: keyof T) {
    const first = a[orderBy];
    const second = b[orderBy];

    if (first instanceof Date && second instanceof Date) {
        return second.getTime() - first.getTime();
    }

    if (second < first) {
        return -1;
    }

    if (second > first) {
        return 1;
    }

    return 0;
}

type Order = "asc" | "desc";

function getComparator<T>(order: Order, orderBy: keyof T) {
    return order === "desc"
        ? (a: T, b: T) => descendingComparator(a, b, orderBy)
        : (a: T, b: T) => -descendingComparator(a, b, orderBy);
}

export interface Column<T> {
    id: keyof T;
    label: string;
    render?: (value: T[keyof T], row: T) => React.ReactNode;
}

interface EnhancedTableHeadProps<T> {
    columns: Column<T>[];
    numSelected: number;
    onRequestSort: (event: React.MouseEvent<unknown>, property: keyof T) => void;
    onSelectAllClick: (event: React.ChangeEvent<HTMLInputElement>) => void;
    order: Order;
    orderBy: keyof T;
    rowCount: number;
}

function EnhancedTableHead<T>(props: EnhancedTableHeadProps<T>) {
    const { columns, numSelected, onRequestSort, onSelectAllClick, order, orderBy, rowCount } = props;
    const createSortHandler = (property: keyof T) => (event: React.MouseEvent<unknown>) => {
        onRequestSort(event, property);
    };

    return (
        <TableHead>
            <TableRow>
                <TableCell padding="checkbox">
                    <Checkbox
                        color="primary"
                        indeterminate={numSelected > 0 && numSelected < rowCount}
                        checked={rowCount > 0 && numSelected === rowCount}
                        onChange={onSelectAllClick}
                        slotProps={{
                            input: { "aria-label": "select all rows" },
                        }}
                    />
                </TableCell>
                {columns.map((col) => (
                    <TableCell key={String(col.id)} sortDirection={orderBy === col.id ? order : false}>
                        <TableSortLabel
                            active={orderBy === col.id}
                            direction={orderBy === col.id ? order : "asc"}
                            onClick={createSortHandler(col.id)}
                        >
                            {col.label}
                            {orderBy === col.id ? (
                                <Box component="span" sx={visuallyHidden}>
                                    {order === "desc" ? "sorted descending" : "sorted ascending"}
                                </Box>
                            ) : null}
                        </TableSortLabel>
                    </TableCell>
                ))}
            </TableRow>
        </TableHead>
    );
}
interface EnhancedTableToolbarProps {
    title: string;
    numSelected: number;
    searchQuery: string;
    onSearchChange: (query: string) => void;
    onDelete: () => void;
}
function EnhancedTableToolbar(props: EnhancedTableToolbarProps) {
    const { title, numSelected, searchQuery, onSearchChange, onDelete } = props;
    return (
        <Toolbar
            sx={[
                {
                    pl: { sm: 2 },
                    pr: { xs: 1, sm: 1 },
                },
                numSelected > 0 && {
                    bgcolor: (theme) => alpha(theme.palette.primary.main, theme.palette.action.activatedOpacity),
                },
            ]}
        >
            {numSelected > 0 ? (
                <Typography
                    variant="subtitle1"
                    component="div"
                    sx={{
                        color: "inherit",
                        flex: "1 1 100%",
                    }}
                >
                    {numSelected} selected
                </Typography>
            ) : (
                <Typography sx={{ flex: "1 1 100%" }} variant="h6" id="tableTitle" component="div">
                    {title}
                </Typography>
            )}
            <Stack direction="row" spacing={2}>
                <TextField
                    placeholder="Search"
                    slotProps={{
                        input: {
                            startAdornment: (
                                <InputAdornment position="start">
                                    <SearchIcon />
                                </InputAdornment>
                            ),
                        },
                    }}
                    value={searchQuery}
                    onChange={(e) => onSearchChange(e.target.value)}
                    variant="standard"
                />
                {numSelected > 0 ? (
                    <Tooltip title="Delete">
                        <IconButton onClick={onDelete}>
                            <DeleteIcon />
                        </IconButton>
                    </Tooltip>
                ) : (
                    <Tooltip title="Filter list">
                        <IconButton>
                            <FilterListIcon />
                        </IconButton>
                    </Tooltip>
                )}
            </Stack>
        </Toolbar>
    );
}

interface EnhancedTableProps<T extends Record<string, unknown>> {
    title: string;
    rows: T[];
    columns: Column<T>[];
    getRowId: (row: T) => string;
    onDelete: (ids: string[]) => void;
    defaultRowsPerPage: number;
}
export default function EnhancedTable<T extends Record<string, unknown>>(props: EnhancedTableProps<T>) {
    const { title, rows, columns, getRowId, onDelete, defaultRowsPerPage = 5 } = props;
    const [order, setOrder] = React.useState<Order>("asc");
    const [orderBy, setOrderBy] = React.useState<keyof T>(columns[0].id ?? ("" as keyof T));
    const [selected, setSelected] = React.useState<string[]>([]);
    const [page, setPage] = React.useState(0);
    const [rowsPerPage, setRowsPerPage] = React.useState(defaultRowsPerPage);
    const [searchQuery, setSearchQuery] = React.useState("");

    const filteredRows = React.useMemo(() => {
        if (!searchQuery.trim()) return rows;
        const q = searchQuery.toLowerCase();
        return rows.filter((row) =>
            columns.some((col) =>
                String(row[col.id] ?? "")
                    .toLowerCase()
                    .includes(q),
            ),
        );
    }, [rows, columns, searchQuery]);

    const handleRequestSort = (_: React.MouseEvent<unknown>, property: keyof T) => {
        // _ represents event
        const isAsc = orderBy === property && order === "asc";
        setOrder(isAsc ? "desc" : "asc");
        setOrderBy(property);
    };

    const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
        if (event.target.checked) {
            setSelected(filteredRows.map(getRowId));
        } else {
            setSelected([]);
        }
    };

    const handleRowClick = (_: React.MouseEvent<unknown>, id: string) => {
        setSelected((prev) => {
            const idx = prev.indexOf(id);
            if (idx === -1) return [...prev, id];
            return prev.filter((v) => v !== id);
        });
    };

    const handleChangePage = (_: unknown, newPage: number) => {
        // _ represents event
        setPage(newPage);
    };

    const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
        setRowsPerPage(parseInt(event.target.value, 10));
        setPage(0);
    };

    const handleSearchChange = (query: string) => {
        // TODO: update search
        setSearchQuery(query);
        setPage(0); // reset to first page on new search
    };

    const handleDelete = () => {
        onDelete(selected);
        setSelected([]);
    };

    // Avoid a layout jump when reaching the last page with empty rows.
    const emptyRows = page > 0 ? Math.max(0, (1 + page) * rowsPerPage - rows.length) : 0;

    const visibleRows = React.useMemo(
        () =>
            [...filteredRows]
                .sort(getComparator(order, orderBy))
                .slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage),
        [order, orderBy, page, rowsPerPage],
    );

    return (
        <Box sx={{ width: "100%" }}>
            <Paper sx={{ width: "100%", mb: 2 }}>
                <EnhancedTableToolbar
                    title={title}
                    numSelected={selected.length}
                    searchQuery={searchQuery}
                    onSearchChange={handleSearchChange}
                    onDelete={handleDelete}
                />
                <TableContainer>
                    <Table sx={{ minWidth: 750 }} aria-labelledby="tableTitle">
                        <EnhancedTableHead<T>
                            columns={columns}
                            numSelected={selected.length}
                            order={order}
                            orderBy={orderBy}
                            rowCount={filteredRows.length}
                            onSelectAllClick={handleSelectAllClick}
                            onRequestSort={handleRequestSort}
                        />
                        <TableBody>
                            {visibleRows.map((row, index) => {
                                const rowId = getRowId(row);
                                const isItemSelected = selected.includes(rowId);
                                const labelId = `enhanced-table-checkbox-${index}`;

                                return (
                                    <TableRow
                                        hover
                                        onClick={(event) => handleRowClick(event, rowId)}
                                        role="checkbox"
                                        aria-checked={isItemSelected}
                                        tabIndex={-1}
                                        key={rowId}
                                        selected={isItemSelected}
                                        sx={{ cursor: "pointer" }}
                                    >
                                        <TableCell padding="checkbox">
                                            <Checkbox
                                                color="primary"
                                                checked={isItemSelected}
                                                slotProps={{
                                                    input: { "aria-labelledby": labelId },
                                                }}
                                            />
                                        </TableCell>

                                        {columns.map((col, colIndex) => {
                                            const value = row[col.id];
                                            const cell = col.render ? col.render(value, row) : String(value ?? "");

                                            return colIndex === 0 ? (
                                                <TableCell key={String(col.id)} component="th" id={labelId} scope="row">
                                                    {cell}
                                                </TableCell>
                                            ) : (
                                                <TableCell key={String(col.id)}>{cell}</TableCell>
                                            );
                                        })}
                                    </TableRow>
                                );
                            })}
                            {emptyRows > 0 && (
                                <TableRow>
                                    <TableCell colSpan={6} />
                                </TableRow>
                            )}
                        </TableBody>
                    </Table>
                </TableContainer>
                <TablePagination
                    rowsPerPageOptions={[5, 10, 25]}
                    component="div"
                    count={rows.length}
                    rowsPerPage={rowsPerPage}
                    page={page}
                    onPageChange={handleChangePage}
                    onRowsPerPageChange={handleChangeRowsPerPage}
                />
            </Paper>
        </Box>
    );
}
