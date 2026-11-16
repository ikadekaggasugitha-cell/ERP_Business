import * as React from 'react';
import { CardSkeleton } from './Skeleton';
import { ErrorState } from './ErrorState';

interface InnerBoundaryProps {
  resetKey?: string;
  children: React.ReactNode;
}

interface InnerBoundaryState {
  error: Error | null;
}

class DomainBoundaryInner extends React.Component<InnerBoundaryProps, InnerBoundaryState> {
  constructor(props: InnerBoundaryProps) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: Error): InnerBoundaryState {
    return { error };
  }

  componentDidUpdate(prevProps: InnerBoundaryProps): void {
    if (prevProps.resetKey !== this.props.resetKey && this.state.error !== null) {
      this.setState({ error: null });
    }
  }

  render() {
    if (this.state.error) {
      return (
        <ErrorState
          error={this.state.error}
          code="MFE_WORKSPACE_ERROR"
          onRetry={() => this.setState({ error: null })}
          className="my-4"
        />
      );
    }
    return this.props.children;
  }
}

export interface DomainErrorBoundaryProps {
  resetKey?: string;
  fallback?: React.ReactNode;
  children: React.ReactNode;
}

/**
 * Option A Modular MFE isolation boundary:
 * isolates domain chunk lazy loading (Suspense) and runtime failures (ErrorBoundary)
 * per workspace without taking down the surrounding portal shell.
 */
export function DomainErrorBoundary({
  resetKey,
  fallback,
  children,
}: DomainErrorBoundaryProps) {
  return (
    <DomainBoundaryInner resetKey={resetKey}>
      <React.Suspense
        fallback={
          fallback ?? (
            <div className="space-y-4">
              <CardSkeleton />
              <CardSkeleton />
            </div>
          )
        }
      >
        {children}
      </React.Suspense>
    </DomainBoundaryInner>
  );
}
