import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import { MORGAN } from '../../../src/data';

describe('DocsView', () => {
  it('renders the heading and a relative link to the generated API reference', () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(screen.getByRole('heading', { level: 2, name: /API documentation/ })).toBeInTheDocument();
    const api = screen.getByRole('link', { name: /Open the API reference/ });
    expect(api).toHaveAttribute('href', './api/');
    const github = screen.getByRole('link', { name: /Source on GitHub/ });
    expect(github).toHaveAttribute('href', MORGAN.repo);
    expect(github).toHaveAttribute('target', '_blank');
  });
});
