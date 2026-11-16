import { createFileRoute } from '@tanstack/react-router';
import { PayrollWorkspace } from '../../features/payroll/PayrollWorkspace';

export const Route = createFileRoute('/_auth/payroll')({
  component: PayrollPage,
});

function PayrollPage() {
  return <PayrollWorkspace />;
}
