import styles from './filterselect.module.css';
import {useId} from 'react';
import {ChevronDown} from 'lucide-react';

export interface FilterOption<T extends string> {
    value: T;
    label: string;
    count?: number;
}

interface FilterSelectProps<T extends string> {
    label: string;
    options: FilterOption<T>[];
    selected: T | null;
    onChange: (value: T | null) => void;
}

const FilterSelect = <T extends string>({label, options, selected, onChange}: FilterSelectProps<T>) =>
{
    const id = useId();

    return (
        <div className={`${styles.field} ${selected ? styles.active : ''}`}>
            <label htmlFor={id} className={styles.label}>{label}</label>
            <div className={styles.control}>
                <select
                    id={id}
                    value={selected ?? ''}
                    onChange={event => onChange(event.target.value === '' ? null : event.target.value as T)}
                >
                    <option value=''>Todos</option>
                    {options.map(option => (
                        <option key={option.value} value={option.value}>
                            {option.count === undefined ? option.label : `${option.label} (${option.count})`}
                        </option>
                    ))}
                </select>
                <ChevronDown size={16} className={styles.chevron} aria-hidden='true'/>
            </div>
        </div>
    );
}

export default FilterSelect;
