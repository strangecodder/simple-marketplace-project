create table product
(
    product_id          uuid default gen_random_uuid() not null
        constraint product_pk
            primary key,
    seller_id           uuid                           not null,
    product_name         varchar                        not null,
    product_description varchar                        not null,
    price               bigint                         not null
);

alter table product
    owner to postgres;

create table product_action
(
    product_action_id uuid default gen_random_uuid() not null
        constraint product_action_pk
            primary key,
    action            varchar                        not null,
    action_value      bigint                         not null,
    action_time       timestamp                      not null
);

-- change_count -> 200
-- change_count -> 180

alter table product_action
    owner to postgres;

