import styles from './pagination.module.css';
import {ChevronLeft, ChevronRight} from 'lucide-react';

interface PaginationProps {
    page: number;
    limit: number;
    total: number;
    itemLabel: string;
    onPageChange: (page: number) => void;
}

const Pagination = ({page, limit, total, itemLabel, onPageChange}: PaginationProps) =>
{
    const totalPages = Math.max(1, Math.ceil(total / limit));
    const from = total === 0 ? 0 : (page - 1) * limit + 1;
    const to = Math.min(page * limit, total);

    return (
        <nav className={styles.pagination} aria-label='Paginación'>
            <span className={styles.summary}>
                Mostrando <strong className='tabular'>{from}–{to}</strong> de <strong className='tabular'>{total}</strong> {itemLabel}
            </span>
            <div className={styles.controls}>
                <button type='button' className={styles.button} onClick={() => onPageChange(page - 1)} disabled={page <= 1} aria-label='Página anterior'>
                    <ChevronLeft size={16}/>
                </button>
                <span className={`${styles.page} tabular`}>{page} / {totalPages}</span>
                <button type='button' className={styles.button} onClick={() => onPageChange(page + 1)} disabled={page >= totalPages} aria-label='Página siguiente'>
                    <ChevronRight size={16}/>
                </button>
            </div>
        </nav>
    );
}

export default Pagination;
