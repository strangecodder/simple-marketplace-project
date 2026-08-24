create table product_order
(
    order_id      uuid default gen_random_uuid() not null
        constraint product_pk
            primary key,
    product_id    uuid                           not null,
    order_status  varchar                        not null,
    product_count varchar                        not null
);

alter table product_order
    owner to postgres;

create table order_action
(
    order_action_id uuid default gen_random_uuid() not null
        constraint product_action_pk
            primary key,
    action_time     timestamp                      not null
);

alter table order_action
    owner to postgres;

create table product_order_count
(
    id               uuid default gen_random_uuid() not null
        constraint product_order_count_pk
            primary key,
    product_order_fk uuid                           not null
        constraint product_order_count_product_order_order_id_fk
            references product_order,
    product_id       uuid                           not null,
    product_count    integer                        not null
);

alter table product_order_count
    owner to postgres;

