import { cn } from '../../lib/utils';

export interface BrandLogoProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const SIZE_CLASSES: Record<NonNullable<BrandLogoProps['size']>, string> = {
  sm: 'h-7 w-7 rounded-md',
  md: 'h-8 w-8 rounded-lg',
  lg: 'h-10 w-10 rounded-xl',
};

/**
 * Vector Prism geometric identity matching FigCraft Design Tokens & Brand Lockup.
 */
export function BrandLogo({ size = 'md', className }: BrandLogoProps) {
  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center justify-center border border-brand-100 bg-slate-50 shadow-2xs',
        'dark:border-brand-900 dark:bg-[#131F38]',
        SIZE_CLASSES[size],
        className
      )}
      aria-hidden="true"
    >
      <svg viewBox="0 0 28 28" className="h-[78%] w-[78%]" fill="none">
        {/* Back-right navy ribbon */}
        <path d="M 14 4.5 L 22.5 10 L 18.5 19 L 14 16 Z" fill="#1E3A8A" />
        {/* Front-left cobalt facet */}
        <path d="M 14 4.5 L 5.5 10 L 9.5 19 L 14 13 Z" fill="#2563EB" />
        {/* Bottom emerald interlocking wedge */}
        <path d="M 9.5 19 L 14 14.5 L 18.5 19 L 14 23.5 Z" fill="#10B981" />
        {/* Crisp center highlight ridge */}
        <path d="M 14 6.2 L 17.5 10.5 L 14 13.5 L 10.5 10.5 Z" fill="#60A5FA" />
      </svg>
    </span>
  );
}

