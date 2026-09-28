import {AlertTriangle} from 'lucide-react';
import Button from '@/components/shared/button/Button.tsx';
import Modal from '@/components/shared/modal/Modal.tsx';

interface ConfirmDialogProps {
    isOpen: boolean;
    title: string;
    message: string;
    confirmLabel: string;
    isLoading?: boolean;
    onConfirm: () => void;
    onCancel: () => void;
}

const ConfirmDialog = ({isOpen, title, message, confirmLabel, isLoading = false, onConfirm, onCancel}: ConfirmDialogProps) =>
{
    return (
        <Modal
            isOpen={isOpen}
            title={title}
            size='sm'
            icon={<AlertTriangle size={20} color='var(--anomaly-real-text)'/>}
            onClose={onCancel}
            footer={
                <>
                    <Button onClick={onCancel} disabled={isLoading}>Cancelar</Button>
                    <Button variant='danger' onClick={onConfirm} isLoading={isLoading}>{confirmLabel}</Button>
                </>
            }
        >
            <p style={{color: 'var(--text-secondary)'}}>{message}</p>
        </Modal>
    );
}

export default ConfirmDialog;
