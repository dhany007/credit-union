"use client";

import { useId, useRef, useState } from "react";
import { X } from "lucide-react";
import { Dialog, DialogContent, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { usePreferences } from "@/components/preferences";
import { localizedError } from "@/lib/api-errors";

export function Modal({ open, onClose, title, description, children, footer, blocked = false, wide = false, scrollable = true }: {
  open: boolean; onClose: () => void; title: string; description?: string;
  children: React.ReactNode; footer?: React.ReactNode; blocked?: boolean; wide?: boolean; scrollable?: boolean;
}) {
  const { t } = usePreferences();
  const descriptionID = useId();
  const previousFocus = useRef<HTMLElement | null>(null);
  return <Dialog open={open} onOpenChange={value => { if (!value && !blocked) onClose(); }}>
    <DialogContent className={wide ? "max-w-4xl" : undefined} aria-describedby={description ? descriptionID : undefined}
      onOpenAutoFocus={() => { previousFocus.current = document.activeElement instanceof HTMLElement ? document.activeElement : null; }}
      onCloseAutoFocus={event => {
        event.preventDefault();
        const previous = previousFocus.current;
        const target = previous?.isConnected && !previous.matches(":disabled") ? previous
          : document.querySelector<HTMLElement>("[data-modal-content] button:not(:disabled)") ?? document.querySelector<HTMLElement>("main button:not(:disabled)");
        target?.focus();
      }}
      onPointerDownOutside={event => event.preventDefault()}
      onEscapeKeyDown={event => { if (blocked) event.preventDefault(); }}>
      <header className="relative shrink-0 border-b px-5 py-4 pr-14">
        <DialogTitle className="text-lg font-semibold">{title}</DialogTitle>
        {description && <DialogDescription id={descriptionID} className="mt-1 text-sm text-muted-foreground">{description}</DialogDescription>}
        <Button type="button" variant="ghost" size="icon" aria-label={t("Tutup")} disabled={blocked} onClick={onClose} className="no-print absolute right-3 top-3"><X aria-hidden="true" size={18}/></Button>
      </header>
      <div className={scrollable ? "min-h-0 overflow-y-auto p-5" : "flex min-h-0 flex-1 flex-col"}>{children}</div>
      {footer && <footer className="no-print flex shrink-0 flex-wrap justify-end gap-3 border-t px-5 py-4">{footer}</footer>}
    </DialogContent>
  </Dialog>;
}

export function Notification({ title, children, onClose }: { title: string; children: React.ReactNode; onClose?: () => void }) {
  const { t } = usePreferences();
  const [open, setOpen] = useState(true);
  function close() { setOpen(false); onClose?.(); }
  return <Modal open={open} onClose={close} title={title} footer={<Button type="button" onClick={close}>{t("Tutup")}</Button>}>{children}</Modal>;
}

export function ErrorNotice({ error, onClose }: { error: string; onClose?: () => void }) {
  const { t, locale } = usePreferences();
  return error ? <Notification key={error} title={t("Tidak dapat melanjutkan")} onClose={onClose}><p role="alert" className="text-sm text-destructive">{localizedError(locale, error)}</p></Notification> : null;
}
