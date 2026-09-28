import {useEffect, useState} from 'react';

export const useDebouncedValue = <T>(value: T, delayMs: number = 300): T =>
{
    const [debounced, setDebounced] = useState(value);

    useEffect(() =>
    {
        const timer = setTimeout(() => setDebounced(value), delayMs);

        return () => clearTimeout(timer);
    }, [value, delayMs]);

    return debounced;
}

export default useDebouncedValue;
