import { IconButton, InputAdornment, TextField } from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";
import ClearIcon from "@mui/icons-material/Clear";
import { memo, useEffect, useState } from "react";

interface SearchFieldProps {
    onDebouncedChange: (value: string) => void;
}

const DEBOUNCE_TIMEOUT = 300;

export default memo(function SearchField({ onDebouncedChange }: SearchFieldProps) {
    const [searchInput, setSearchInput] = useState<string>("");

    const handleClear = () => {
        setSearchInput("");
    };

    useEffect(() => {
        // set a delay until the user stops entering any search input
        const timer = setTimeout(() => onDebouncedChange(searchInput), DEBOUNCE_TIMEOUT);
        return () => clearTimeout(timer);
    }, [searchInput, onDebouncedChange]);

    return (
        <TextField
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder="Search body..."
            size="small"
            slotProps={{
                input: {
                    startAdornment: (
                        <InputAdornment position="start">
                            <SearchIcon color="action" />
                        </InputAdornment>
                    ),
                    endAdornment: searchInput && (
                        <InputAdornment position="end">
                            <IconButton onClick={handleClear} edge="end" size="small">
                                <ClearIcon />
                            </IconButton>
                        </InputAdornment>
                    ),
                },
            }}
        />
    );
});
