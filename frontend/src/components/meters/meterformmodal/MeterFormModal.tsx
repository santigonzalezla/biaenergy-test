import styles from './meterformmodal.module.css';
import {useState, type FormEvent} from 'react';
import {Gauge} from 'lucide-react';
import Button from '@/components/shared/button/Button.tsx';
import FormField from '@/components/shared/formfield/FormField.tsx';
import Modal from '@/components/shared/modal/Modal.tsx';
import type {CreateMeterRequest, Meter, UpdateMeterRequest} from '@/interfaces/interfaces.ts';
import {ApiError, apiRequest} from '@/lib/api.ts';

type Field = 'code' | 'name' | 'location' | 'sector' | 'nominalVoltage' | 'maxCurrent' | 'contractedPowerKw';

type FormValues = Record<Field, string>;

type FieldErrors = Partial<Record<Field, string>>;

interface MeterFormModalProps {
    isOpen: boolean;
    meter?: Meter | null;
    onClose: () => void;
    onSaved: (meter: Meter) => void;
}

const CODE_PATTERN = /^[A-Z0-9-]{2,20}$/;

const FIELD_MESSAGES: Record<Field, string> = {
    code: 'Usa de 2 a 20 caracteres: letras, números o guion.',
    name: 'El nombre es obligatorio (máximo 120 caracteres).',
    location: 'Ubicación no válida.',
    sector: 'Sector no válido.',
    nominalVoltage: 'Debe ser un número mayor que cero.',
    maxCurrent: 'Debe ser un número mayor que cero.',
    contractedPowerKw: 'Debe ser un número mayor que cero.',
};

const NUMERIC_FIELDS: Field[] = ['nominalVoltage', 'maxCurrent', 'contractedPowerKw'];

const toText = (value: number | null | undefined) => (value === null || value === undefined ? '' : String(value));

const initialValues = (meter?: Meter | null): FormValues => ({
    code: meter?.code ?? '',
    name: meter?.name ?? '',
    location: meter?.location ?? '',
    sector: meter?.sector ?? '',
    nominalVoltage: toText(meter?.nominalVoltage ?? 220),
    maxCurrent: toText(meter?.maxCurrent),
    contractedPowerKw: toText(meter?.contractedPowerKw),
});

const toNumber = (value: string) => (value.trim() === '' ? undefined : Number(value.replace(',', '.')));

const validate = (values: FormValues, isEdit: boolean): FieldErrors =>
{
    const errors: FieldErrors = {};

    if (!isEdit && !CODE_PATTERN.test(values.code.trim().toUpperCase())) errors.code = FIELD_MESSAGES.code;

    const name = values.name.trim();

    if (name === '' || name.length > 120) errors.name = FIELD_MESSAGES.name;

    NUMERIC_FIELDS.forEach(field =>
    {
        const number = toNumber(values[field]);

        if (number !== undefined && (!Number.isFinite(number) || number <= 0)) errors[field] = FIELD_MESSAGES[field];
    });

    return errors;
}

const MeterFormModal = ({isOpen, meter, onClose, onSaved}: MeterFormModalProps) =>
{
    const isEdit = Boolean(meter);
    const [values, setValues] = useState<FormValues>(() => initialValues(meter));
    const [errors, setErrors] = useState<FieldErrors>({});
    const [formError, setFormError] = useState<string | null>(null);
    const [isSaving, setIsSaving] = useState(false);

    const update = (field: Field) => (event: {target: {value: string}}) =>
    {
        setValues(previous => ({...previous, [field]: event.target.value}));
        setErrors(previous => ({...previous, [field]: undefined}));
    }

    const handleSubmit = async (event: FormEvent<HTMLFormElement>) =>
    {
        event.preventDefault();

        const clientErrors = validate(values, isEdit);
        setErrors(clientErrors);
        setFormError(null);

        if (Object.keys(clientErrors).length > 0) return;

        const shared = {
            name: values.name.trim(),
            location: values.location.trim(),
            sector: values.sector.trim(),
            nominalVoltage: toNumber(values.nominalVoltage),
            maxCurrent: toNumber(values.maxCurrent),
            contractedPowerKw: toNumber(values.contractedPowerKw),
        };

        setIsSaving(true);

        try
        {
            const saved = meter
                ? await apiRequest<Meter>(`/meters/${meter.id}`, {method: 'PATCH', body: shared satisfies UpdateMeterRequest})
                : await apiRequest<Meter>('/meters', {method: 'POST', body: {...shared, code: values.code.trim().toUpperCase()} satisfies CreateMeterRequest});

            onSaved(saved);
        }
        catch (caught)
        {
            const apiError = ApiError.from(caught);

            if (apiError.code === 'METER_DUPLICATE_CODE') setErrors({code: apiError.message});
            else if (apiError.code === 'VALIDATION_ERROR' && apiError.details && !Array.isArray(apiError.details))
            {
                const serverErrors: FieldErrors = {};
                Object.keys(apiError.details).forEach(field => serverErrors[field as Field] = FIELD_MESSAGES[field as Field] ?? apiError.message);
                setErrors(serverErrors);
            }
            else setFormError(apiError.message);
        }
        finally
        {
            setIsSaving(false);
        }
    }

    return (
        <Modal
            isOpen={isOpen}
            title={isEdit ? `Editar medidor · ${meter?.code}` : 'Nuevo medidor'}
            description={isEdit ? 'Actualiza los datos del medidor.' : 'Registra un punto de medición para incluirlo en los análisis.'}
            icon={<Gauge size={20}/>}
            onClose={onClose}
            footer={
                <>
                    <Button onClick={onClose} disabled={isSaving}>Cancelar</Button>
                    <Button variant='primary' type='submit' form='meter-form' isLoading={isSaving}>
                        {isEdit ? 'Guardar cambios' : 'Crear medidor'}
                    </Button>
                </>
            }
        >
            <form id='meter-form' className={styles.form} onSubmit={handleSubmit} noValidate>
                {formError && <p className={styles.formError} role='alert'>{formError}</p>}

                <FormField id='meter-code' label='Código' hint='Ej. M-113. No se puede cambiar después.' error={errors.code}>
                    <input id='meter-code' value={values.code} onChange={update('code')} disabled={isEdit} className='mono' aria-invalid={Boolean(errors.code)}/>
                </FormField>

                <FormField id='meter-name' label='Nombre' error={errors.name}>
                    <input id='meter-name' value={values.name} onChange={update('name')} aria-invalid={Boolean(errors.name)}/>
                </FormField>

                <FormField id='meter-location' label='Ubicación' error={errors.location}>
                    <input id='meter-location' value={values.location} onChange={update('location')} placeholder='Ej. Planta norte · Nave C'/>
                </FormField>

                <FormField id='meter-sector' label='Sector' error={errors.sector}>
                    <input id='meter-sector' value={values.sector} onChange={update('sector')} placeholder='Ej. INDUSTRIAL'/>
                </FormField>

                <FormField id='meter-voltage' label='Voltaje nominal (V)' error={errors.nominalVoltage}>
                    <input id='meter-voltage' inputMode='decimal' value={values.nominalVoltage} onChange={update('nominalVoltage')} aria-invalid={Boolean(errors.nominalVoltage)}/>
                </FormField>

                <FormField id='meter-current' label='Corriente máxima (A)' hint='Opcional' error={errors.maxCurrent}>
                    <input id='meter-current' inputMode='decimal' value={values.maxCurrent} onChange={update('maxCurrent')} aria-invalid={Boolean(errors.maxCurrent)}/>
                </FormField>

                <FormField id='meter-power' label='Potencia contratada (kW)' hint='Opcional' error={errors.contractedPowerKw}>
                    <input id='meter-power' inputMode='decimal' value={values.contractedPowerKw} onChange={update('contractedPowerKw')} aria-invalid={Boolean(errors.contractedPowerKw)}/>
                </FormField>
            </form>
        </Modal>
    );
}

export default MeterFormModal;
