-- +goose Up
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_canal_check;
ALTER TABLE payments
    ADD CONSTRAINT payments_canal_check CHECK (canal IN ('efectivo', 'transferencia', 'deposito', 'tarjeta', 'cajero', 'mercadopago', 'otros'));

-- +goose Down
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_canal_check;
ALTER TABLE payments
    ADD CONSTRAINT payments_canal_check CHECK (canal IN ('efectivo', 'transferencia', 'deposito', 'tarjeta', 'cajero', 'otros'));
