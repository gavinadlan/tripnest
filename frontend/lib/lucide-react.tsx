import React from 'react';

type IconProps = React.SVGProps<SVGSVGElement>;

function icon() {
  return function Icon(props: IconProps) {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true" {...props}>
        <path d="M4 12h16" />
        <path d="M12 4v16" />
      </svg>
    );
  };
}

export const PlaneTakeoff = icon();
export const Mail = icon();
export const Lock = icon();
export const LogIn = icon();
export const User = icon();
export const CheckCircle2 = icon();
export const LogOut = icon();
export const Menu = icon();
export const X = icon();
export const BarChart3 = icon();
export const Boxes = icon();
export const CreditCard = icon();
export const NotebookTabs = icon();
export const CalendarDays = icon();
export const MapPin = icon();
export const Users = icon();
export const Tag = icon();
export const Search = icon();
export const DollarSign = icon();
export const Calendar = icon();
export const Loader2 = icon();
