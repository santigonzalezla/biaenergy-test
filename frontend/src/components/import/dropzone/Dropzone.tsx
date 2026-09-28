import styles from './dropzone.module.css';
import {useId, useRef, useState, type DragEvent} from 'react';
import {FileSpreadsheet, UploadCloud, X} from 'lucide-react';
import {formatNumber} from '@/lib/format.ts';

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

interface DropzoneProps {
    label: string;
    file: File | null;
    disabled?: boolean;
    onFileChange: (file: File | null) => void;
    onReject: (message: string) => void;
}

const validate = (file: File): string | null =>
{
    if (!file.name.toLowerCase().endsWith('.csv')) return 'Selecciona un archivo con extensión .csv.';
    if (file.size === 0) return 'El archivo está vacío.';
    if (file.size > MAX_UPLOAD_BYTES) return 'El archivo supera el máximo de 10 MB.';

    return null;
}

const Dropzone = ({label, file, disabled = false, onFileChange, onReject}: DropzoneProps) =>
{
    const inputId = useId();
    const inputRef = useRef<HTMLInputElement>(null);
    const [isDragging, setIsDragging] = useState(false);

    const accept = (candidate: File | undefined) =>
    {
        if (!candidate) return;

        const error = validate(candidate);

        if (error)
        {
            onReject(error);
            return;
        }

        onFileChange(candidate);
    }

    const handleDrop = (event: DragEvent<HTMLLabelElement>) =>
    {
        event.preventDefault();
        setIsDragging(false);

        if (!disabled) accept(event.dataTransfer.files[0]);
    }

    const clear = () =>
    {
        onFileChange(null);

        if (inputRef.current) inputRef.current.value = '';
    }

    if (file)
    {
        return (
            <div className={styles.selected}>
                <FileSpreadsheet size={22} className={styles.fileIcon}/>
                <div className={styles.fileInfo}>
                    <span className={styles.fileName}>{file.name}</span>
                    <span className={styles.fileSize}>{formatNumber(file.size / 1024, 1)} KB</span>
                </div>
                <button type='button' className={styles.remove} onClick={clear} disabled={disabled} aria-label={`Quitar ${file.name}`}>
                    <X size={16}/>
                </button>
            </div>
        );
    }

    return (
        <label
            htmlFor={inputId}
            className={`${styles.zone} ${isDragging ? styles.dragging : ''} ${disabled ? styles.disabled : ''}`}
            onDragOver={event =>
            {
                event.preventDefault();
                setIsDragging(true);
            }}
            onDragLeave={() => setIsDragging(false)}
            onDrop={handleDrop}
        >
            <UploadCloud size={28} className={styles.icon}/>
            <span className={styles.title}>Arrastra el CSV aquí o <strong>haz clic para elegirlo</strong></span>
            <span className={styles.hint}>Solo .csv · máximo 10 MB</span>
            <input
                ref={inputRef}
                id={inputId}
                type='file'
                accept='.csv,text/csv'
                className={styles.input}
                aria-label={label}
                disabled={disabled}
                onChange={event => accept(event.target.files?.[0])}
            />
        </label>
    );
}

export default Dropzone;
