-- Drop tables if they already exist (to allow re-runs)
DROP TABLE IF EXISTS public.top_selling_books CASCADE;
DROP TABLE IF EXISTS public.sales_reports CASCADE;
DROP TABLE IF EXISTS public.reviews CASCADE;
DROP TABLE IF EXISTS public.order_items CASCADE;
DROP TABLE IF EXISTS public.orders CASCADE;
DROP TABLE IF EXISTS public.customers CASCADE;
DROP TABLE IF EXISTS public.books CASCADE;
DROP TABLE IF EXISTS public.authors CASCADE;

-- Table: public.authors
CREATE TABLE IF NOT EXISTS public.authors (
    id         integer NOT NULL DEFAULT nextval('authors_id_seq'::regclass),
    first_name text    NOT NULL,
    last_name  text    NOT NULL,
    bio        text,
    CONSTRAINT authors_pkey PRIMARY KEY (id)
)
TABLESPACE pg_default;
ALTER TABLE public.authors OWNER TO postgres;

-- Table: public.books
CREATE TABLE IF NOT EXISTS public.books (
    id            integer     NOT NULL DEFAULT nextval('books_id_seq'::regclass),
    title         text        NOT NULL,
    author_id     integer     NOT NULL,
    genres        text[],
    published_at  timestamp   NOT NULL,
    price         numeric(10,2) NOT NULL,
    stock         integer     NOT NULL,
    review_stats  jsonb,
    CONSTRAINT books_pkey PRIMARY KEY (id)
)
TABLESPACE pg_default;
ALTER TABLE public.books OWNER TO postgres;

-- Table: public.customers
CREATE TABLE IF NOT EXISTS public.customers (
    id           integer     NOT NULL DEFAULT nextval('customers_id_seq'::regclass),
    name         text        NOT NULL,
    email        text        NOT NULL,
    street       text,
    city         text,
    state        text,
    postal_code  text,
    country      text,
    created_at   timestamp   NOT NULL,
    username     varchar(255),
    password     varchar(255),
    CONSTRAINT customers_pkey PRIMARY KEY (id),
    CONSTRAINT customers_email_key UNIQUE (email)
)
TABLESPACE pg_default;
ALTER TABLE public.customers OWNER TO postgres;

-- Table: public.orders
CREATE TABLE IF NOT EXISTS public.orders (
    id           integer      NOT NULL DEFAULT nextval('orders_id_seq'::regclass),
    customer_id  integer      NOT NULL,
    total_price  numeric(10,2) NOT NULL,
    created_at   timestamp    NOT NULL,
    status       text         NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (id)
)
TABLESPACE pg_default;
ALTER TABLE public.orders OWNER TO postgres;

-- Table: public.order_items
CREATE TABLE IF NOT EXISTS public.order_items (
    id        integer NOT NULL DEFAULT nextval('order_items_id_seq'::regclass),
    order_id  integer NOT NULL,
    book_id   integer NOT NULL,
    quantity  integer NOT NULL,
    CONSTRAINT order_items_pkey PRIMARY KEY (id),
    CONSTRAINT order_items_order_id_fkey FOREIGN KEY (order_id)
        REFERENCES public.orders (id) ON UPDATE NO ACTION ON DELETE CASCADE
)
TABLESPACE pg_default;
ALTER TABLE public.order_items OWNER TO postgres;

-- Table: public.reviews
CREATE TABLE IF NOT EXISTS public.reviews (
    id               integer   NOT NULL DEFAULT nextval('reviews_id_seq'::regclass),
    book_id          integer   NOT NULL,
    customer_id      integer,
    rating           integer   NOT NULL,
    review_text      text      NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT reviews_pkey PRIMARY KEY (id),
    CONSTRAINT reviews_book_id_fkey FOREIGN KEY (book_id)
        REFERENCES public.books (id) ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT reviews_customer_id_fkey FOREIGN KEY (customer_id)
        REFERENCES public.customers (id) ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT reviews_rating_check CHECK (rating >= 1 AND rating <= 5)
)
TABLESPACE pg_default;
ALTER TABLE public.reviews OWNER TO postgres;

-- Table: public.sales_reports
CREATE TABLE IF NOT EXISTS public.sales_reports (
    id                integer      NOT NULL DEFAULT nextval('sales_reports_id_seq'::regclass),
    "timestamp"       timestamp    NOT NULL,
    total_revenue     numeric(10,2) NOT NULL,
    total_orders      integer      NOT NULL,
    successful_orders integer      NOT NULL,
    pending_orders    integer      NOT NULL,
    CONSTRAINT sales_reports_pkey PRIMARY KEY (id)
)
TABLESPACE pg_default;
ALTER TABLE public.sales_reports OWNER TO postgres;

-- Index on sales_reports.timestamp
CREATE INDEX IF NOT EXISTS idx_sales_reports_timestamp
    ON public.sales_reports ("timestamp" ASC NULLS LAST)
    TABLESPACE pg_default;

-- Table: public.top_selling_books
CREATE TABLE IF NOT EXISTS public.top_selling_books (
    id                    integer      NOT NULL DEFAULT nextval('top_selling_books_id_seq'::regclass),
    sales_report_id       integer      NOT NULL,
    book_id               integer      NOT NULL,
    quantity_sold         integer      NOT NULL,
    total_revenue         numeric(10,2) NOT NULL DEFAULT 0,
    book_title            text         NOT NULL,
    book_price            numeric(10,2) NOT NULL,
    CONSTRAINT top_selling_books_pkey PRIMARY KEY (id),
    CONSTRAINT top_selling_books_book_id_fkey FOREIGN KEY (book_id)
        REFERENCES public.books (id) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT top_selling_books_sales_report_id_fkey FOREIGN KEY (sales_report_id)
        REFERENCES public.sales_reports (id) ON UPDATE NO ACTION ON DELETE CASCADE
)
TABLESPACE pg_default;
ALTER TABLE public.top_selling_books OWNER TO postgres;
