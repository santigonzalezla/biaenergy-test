import styles from './filterpills.module.css';

export interface FilterOption<T extends string> {
    value: T;
    label: string;
    count?: number;
}

interface FilterPillsProps<T extends string> {
    label: string;
    options: FilterOption<T>[];
    selected: T | null;
    onChange: (value: T | null) => void;
}

const FilterPills = <T extends string>({label, options, selected, onChange}: FilterPillsProps<T>) =>
{
    return (
        <div className={styles.group} role='group' aria-label={label}>
            <span className={styles.label}>{label}</span>
            <div className={styles.pills}>
                <button type='button' className={`${styles.pill} ${selected === null ? styles.active : ''}`} aria-pressed={selected === null} onClick={() => onChange(null)}>
                    Todos
                </button>
                {options.map(option => (
                    <button
                        key={option.value}
                        type='button'
                        className={`${styles.pill} ${selected === option.value ? styles.active : ''}`}
                        aria-pressed={selected === option.value}
                        aria-label={option.count === undefined ? option.label : `${option.label} (${option.count})`}
                        onClick={() => onChange(selected === option.value ? null : option.value)}
                    >
                        {option.label}
                        {option.count !== undefined && <span className={`${styles.count} tabular`}>{option.count}</span>}
                    </button>
                ))}
            </div>
        </div>
    );
}

export default FilterPills;
