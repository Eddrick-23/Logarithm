import { Box, Stack, Chip, FormGroup, FormControlLabel, Checkbox, Typography } from "@mui/material";
import React from "react";

interface ServiceFiltersProps {
    services: string[];
    serviceColours: Record<string, string>;
    hiddenServices: Set<string>;
    setHiddenServices: React.Dispatch<React.SetStateAction<Set<string>>>;
}

export const ServiceFilters = React.memo(function ServiceFilters({
    services,
    serviceColours,
    hiddenServices,
    setHiddenServices,
}: ServiceFiltersProps) {
    const toggleService = (service: string) => {
        setHiddenServices((prev) => {
            const currServices = new Set(prev);
            if (currServices.has(service)) {
                currServices.delete(service);
            } else {
                currServices.add(service);
            }
            return currServices;
        });
    };

    const allSelected = services.every((service) => !hiddenServices.has(service));
    const toggleAll = () => {
        setHiddenServices(allSelected ? new Set(services) : new Set());
    };

    return (
        <Box sx={{ my: 2 }}>
            <Stack direction="row" sx={{ flexWrap: "wrap", gap: 2.5, alignItems: "center" }}>
                <Chip
                    label="All"
                    size="small"
                    variant={allSelected ? "filled" : "outlined"}
                    onClick={toggleAll}
                    sx={{ fontWeight: 600 }}
                />
                <Stack direction="row" sx={{ flexWrap: "wrap" }}>
                    {services.map((service) => (
                        <FormControlLabel
                            key={service}
                            control={
                                <Checkbox
                                    size="small"
                                    checked={!hiddenServices.has(service)}
                                    onChange={() => toggleService(service)}
                                    sx={{
                                        color: serviceColours[service],
                                        "&.Mui-checked": {
                                            color: serviceColours[service],
                                        },
                                    }}
                                />
                            }
                            label={
                                <Typography variant="body2" noWrap>
                                    {service}
                                </Typography>
                            }
                        />
                    ))}
                </Stack>
            </Stack>
        </Box>
    );
});
