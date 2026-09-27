import {render, screen} from '@testing-library/react';
import App from '@/App.tsx';

describe('App', () =>
{
    it('renders the application name', () =>
    {
        render(<App/>);

        expect(screen.getByRole('heading', {name: 'Bia Energy · Anomaly Center'})).toBeInTheDocument();
    });
});
